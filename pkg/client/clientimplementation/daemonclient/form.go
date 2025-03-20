package daemonclient

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/ghodss/yaml"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/cmd/pro/provider/list"
	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/loft-sh/devpod/pkg/platform"
	platformclient "github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/labels"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createInstanceInteractive(ctx context.Context, baseClient platformclient.Client, id, uid, source, picture string, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	formCtx, cancelForm := context.WithCancel(ctx)
	defer cancelForm()

	var selectedCluster *managementv1.Cluster
	var selectedProject *managementv1.Project
	var selectedTemplate *managementv1.DevPodWorkspaceTemplate
	selectedTemplateVersion := ""
	projectOptions, err := projectOptions(ctx, baseClient)
	if err != nil {
		return nil, err
	}
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[*managementv1.Project]().
				Title("Project").
				Options(projectOptions...).
				Value(&selectedProject),
			huh.NewSelect[*managementv1.Cluster]().
				Title("Cluster").
				OptionsFunc(func() []huh.Option[*managementv1.Cluster] {
					return getClusterOptions(ctx, baseClient, selectedProject, cancelForm, log)
				}, &selectedProject).
				Value(&selectedCluster).
				WithHeight(5),
			huh.NewSelect[*managementv1.DevPodWorkspaceTemplate]().
				Title("Template").
				OptionsFunc(func() []huh.Option[*managementv1.DevPodWorkspaceTemplate] {
					return getTemplateOptions(ctx, baseClient, selectedProject, cancelForm, log)
				}, &selectedProject).
				Value(&selectedTemplate),
			huh.NewSelect[string]().
				Title("Template Version").
				OptionsFunc(func() []huh.Option[string] {
					return getTemplateVersionOptions(selectedTemplate)
				}, &selectedTemplate).
				Value(&selectedTemplateVersion).
				WithHeight(8),
		),
	).RunWithContext(formCtx)
	if err != nil {
		return nil, err
	}

	parameters := selectedTemplate.Spec.Parameters
	if len(selectedTemplate.GetVersions()) > 0 {
		parameters, err = list.GetTemplateParameters(selectedTemplate, selectedTemplateVersion)
		if err != nil {
			return nil, err
		}
	}

	renderedParameters := ""
	if len(parameters) > 0 {
		fieldParameters := prepareParameters(parameters)
		err = huh.NewForm(
			huh.NewGroup(parameterFields(fieldParameters)...),
		).RunWithContext(formCtx)
		if err != nil {
			return nil, err
		}

		renderedParameters, err = renderParameters(fieldParameters)
		if err != nil {
			return nil, err
		}
	}

	instance := &managementv1.DevPodWorkspaceInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: encoding.SafeConcatNameMax([]string{id}, 53) + "-",
			Namespace:    project.ProjectNamespace(selectedProject.GetName()),
			Labels: map[string]string{
				storagev1.DevPodWorkspaceIDLabel:  id,
				storagev1.DevPodWorkspaceUIDLabel: uid,
				labels.ProjectLabel:               selectedProject.GetName(),
			},
			Annotations: map[string]string{
				storagev1.DevPodWorkspacePictureAnnotation: picture,
				storagev1.DevPodWorkspaceSourceAnnotation:  source,
			},
		},
		Spec: managementv1.DevPodWorkspaceInstanceSpec{
			DevPodWorkspaceInstanceSpec: storagev1.DevPodWorkspaceInstanceSpec{
				DisplayName: id,
				TemplateRef: &storagev1.TemplateRef{
					Name:    selectedTemplate.GetName(),
					Version: selectedTemplateVersion,
				},
				Target: storagev1.WorkspaceTarget{
					Cluster: &storagev1.WorkspaceTargetName{
						Name: selectedCluster.GetName(),
					},
				},
				Parameters: renderedParameters,
			},
		},
	}

	return instance, nil
}

func updateInstanceInteractive(ctx context.Context, baseClient platformclient.Client, instance *managementv1.DevPodWorkspaceInstance, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	formCtx, cancelForm := context.WithCancel(ctx)
	defer cancelForm()

	projectName := project.ProjectFromNamespace(instance.GetNamespace())
	projectTemplates, err := list.Templates(ctx, baseClient, projectName)
	if err != nil {
		return nil, err
	}
	var selectedTemplate *managementv1.DevPodWorkspaceTemplate
	templateOptions := []TemplateOption{}
	for _, template := range projectTemplates.DevPodWorkspaceTemplates {
		t := &template
		templateOptions = append(templateOptions, huh.Option[*managementv1.DevPodWorkspaceTemplate]{
			Key:   platform.DisplayName(template.GetName(), template.Spec.DisplayName),
			Value: t,
		})

		if instance.Spec.TemplateRef != nil && instance.Spec.TemplateRef.Name == template.GetName() {
			selectedTemplate = t
		}
	}
	if selectedTemplate == nil {
		return nil, fmt.Errorf("template not found: %#v", instance.Spec.TemplateRef)
	}

	var selectedTemplateVersion string
	if instance.Spec.TemplateRef != nil {
		selectedTemplateVersion = instance.Spec.TemplateRef.Version
	}

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[*managementv1.DevPodWorkspaceTemplate]().
				Title("Template").
				Options(templateOptions...).
				Value(&selectedTemplate),
			huh.NewSelect[string]().
				Title("Template Version").
				OptionsFunc(func() []huh.Option[string] {
					return getTemplateVersionOptions(selectedTemplate)
				}, &selectedTemplate).
				Value(&selectedTemplateVersion).
				WithHeight(8),
		),
	).RunWithContext(formCtx)
	if err != nil {
		return nil, err
	}

	parameters := selectedTemplate.Spec.Parameters
	if len(selectedTemplate.GetVersions()) > 0 {
		parameters, err = list.GetTemplateParameters(selectedTemplate, selectedTemplateVersion)
		if err != nil {
			return nil, err
		}
	}

	renderedParameters := ""
	if len(parameters) > 0 {
		tRef := instance.Spec.TemplateRef
		var existingParameters map[string]interface{}
		if tRef != nil && tRef.Name == selectedTemplate.GetName() && tRef.Version == selectedTemplateVersion {
			existingParameters = map[string]interface{}{}
			err = yaml.Unmarshal([]byte(instance.Spec.Parameters), &existingParameters)
			if err != nil {
				return nil, err
			}
		}

		fieldParameters := []*FieldParameter{}
		// reuse existing parameters as starting point
		for _, p := range parameters {
			var value interface{} = p.DefaultValue
			if existingParameters != nil {
				value = getDeepValue(existingParameters, p.Variable)
			}
			fieldParameter := FieldParameter{AppParameter: p}

			if p.Type == "boolean" && value != nil {
				v, err := strconv.ParseBool(value.(string))
				if err == nil {
					fieldParameter.BoolValue = v
				}
			} else {
				if value != nil {
					fieldParameter.StringValue = value.(string)
				} else {
					fieldParameter.StringValue = p.DefaultValue
				}
			}
			fieldParameters = append(fieldParameters, &fieldParameter)
		}

		err = huh.NewForm(
			huh.NewGroup(parameterFields(fieldParameters)...),
		).RunWithContext(formCtx)
		if err != nil {
			return nil, err
		}

		renderedParameters, err = renderParameters(fieldParameters)
		if err != nil {
			return nil, err
		}
	}

	newInstance := instance.DeepCopy()
	// template
	if instance.Spec.TemplateRef != nil && instance.Spec.TemplateRef.Name != selectedTemplate.GetName() {
		newInstance.Spec.TemplateRef.Name = selectedTemplate.GetName()
	}
	// version
	if instance.Spec.TemplateRef != nil && instance.Spec.TemplateRef.Version != selectedTemplateVersion {
		newInstance.Spec.TemplateRef.Version = selectedTemplateVersion
	}
	// parameters
	if instance.Spec.Parameters != renderedParameters {
		newInstance.Spec.Parameters = renderedParameters
	}

	return newInstance, nil
}

type ProjectOption = huh.Option[*managementv1.Project]
type TemplateOption = huh.Option[*managementv1.DevPodWorkspaceTemplate]
type CancelFunc = func()

var latestTemplateVersion = huh.Option[string]{
	Key:   "latest",
	Value: "",
}

func projectOptions(ctx context.Context, client platformclient.Client) ([]ProjectOption, error) {
	projects, err := list.Projects(ctx, client)
	if err != nil {
		return nil, err
	}
	projectOptions := []ProjectOption{}
	for _, project := range projects.Items {
		p := &project
		projectOptions = append(projectOptions, ProjectOption{
			Key:   platform.DisplayName(project.GetName(), project.Spec.DisplayName),
			Value: p,
		})
	}

	return projectOptions, nil
}

func getClusterOptions(ctx context.Context, client platformclient.Client, project *managementv1.Project, cancel CancelFunc, log log.Logger) []huh.Option[*managementv1.Cluster] {
	opts := []huh.Option[*managementv1.Cluster]{}
	if project == nil {
		return opts
	}

	clusters, err := list.Clusters(ctx, client, project.GetName())
	if err != nil {
		log.Error(err)
		cancel()

		return nil
	}
	for _, cluster := range clusters.Clusters {
		r := &cluster
		opts = append(opts, huh.Option[*managementv1.Cluster]{
			Key:   platform.DisplayName(cluster.GetName(), cluster.Spec.DisplayName),
			Value: r,
		})
	}

	return opts
}

func getTemplateOptions(ctx context.Context, client platformclient.Client, project *managementv1.Project, cancel CancelFunc, log log.Logger) []huh.Option[*managementv1.DevPodWorkspaceTemplate] {
	opts := []huh.Option[*managementv1.DevPodWorkspaceTemplate]{}
	if project == nil {
		return opts
	}

	templates, err := list.Templates(ctx, client, project.GetName())
	if err != nil {
		log.Error(err)
		cancel()

		return nil
	}

	var defaultOpt huh.Option[*managementv1.DevPodWorkspaceTemplate]
	for _, template := range templates.DevPodWorkspaceTemplates {
		t := &template
		opt := huh.Option[*managementv1.DevPodWorkspaceTemplate]{
			Key:   platform.DisplayName(template.GetName(), template.Spec.DisplayName),
			Value: t,
		}
		if t.GetName() == templates.DefaultDevPodWorkspaceTemplate {
			defaultOpt = opt
			continue
		}
		opts = append(opts, opt)
	}
	if defaultOpt.Key != "" {
		// make sure the default template is the first
		opts = slices.Insert(opts, 0, defaultOpt)
	}

	return opts
}

func getTemplateVersionOptions(template *managementv1.DevPodWorkspaceTemplate) []huh.Option[string] {
	opts := []huh.Option[string]{latestTemplateVersion}
	if template == nil {
		return opts
	}

	for _, version := range template.GetVersions() {
		opts = append(opts, huh.Option[string]{
			Key:   version.GetVersion(),
			Value: version.GetVersion(),
		})
	}

	return opts
}

type FieldParameter struct {
	storagev1.AppParameter

	StringValue string
	BoolValue   bool
}

func prepareParameters(parameters []storagev1.AppParameter) []*FieldParameter {
	retParams := []*FieldParameter{}
	for _, p := range parameters {
		fieldParameter := FieldParameter{AppParameter: p}
		if p.Type == "boolean" {
			v, err := strconv.ParseBool(p.DefaultValue)
			if err == nil {
				fieldParameter.BoolValue = v
			}
		} else {
			fieldParameter.StringValue = p.DefaultValue
		}

		retParams = append(retParams, &fieldParameter)
	}

	return retParams
}

func parameterFields(fieldParameters []*FieldParameter) []huh.Field {
	fields := []huh.Field{}
	for _, param := range fieldParameters {
		title := param.Label
		if title == "" {
			title = param.Variable
		}
		paramType := param.Type
		if paramType == "" {
			paramType = "string"
		}

		var field huh.Field
		switch paramType {
		case "multiline":
			field = huh.NewText().
				Title(title).
				Description(param.Description).
				Value(&param.StringValue)
		case "password":
			fallthrough
		case "number":
			fallthrough
		case "string":
			// display a select field if param has options
			if len(param.Options) > 0 {
				opts := []huh.Option[string]{}
				for _, o := range param.Options {
					huhOption := huh.Option[string]{
						Key:   o,
						Value: o,
					}
					if o == param.DefaultValue {
						huhOption = huhOption.Selected(true)
					}
					opts = append(opts, huhOption)
				}
				field = huh.NewSelect[string]().
					Title(title).
					Options(opts...).
					Value(&param.StringValue)
			} else {
				input := huh.NewInput().Title(title).
					Description(param.Description).
					Value(&param.StringValue)

				if param.Type == "password" {
					input.EchoMode(huh.EchoModePassword)
				}
				if param.Type == "number" {
					input.Validate(func(s string) error {
						_, err := strconv.ParseFloat(s, 64)
						return err
					})
				}
				field = input
			}
		case "boolean":
			field = huh.NewConfirm().
				Title(title).
				Description(param.Description).
				Value(&param.BoolValue)
		}

		fields = append(fields, field)
	}

	return fields
}

func renderParameters(fieldParameters []*FieldParameter) (string, error) {
	p := map[string]interface{}{}
	for _, fp := range fieldParameters {
		if fp.StringValue != "" {
			p[fp.Variable] = fp.StringValue
		} else if fp.BoolValue {
			p[fp.Variable] = strconv.FormatBool(fp.BoolValue)
		}
	}

	rawParameters, err := yaml.Marshal(p)
	if err != nil {
		return "", err
	}

	return string(rawParameters), nil
}

func getDeepValue(parameters interface{}, path string) interface{} {
	if parameters == nil {
		return nil
	}

	pathSegments := strings.Split(path, ".")
	switch t := parameters.(type) {
	case map[string]interface{}:
		val, ok := t[pathSegments[0]]
		if !ok {
			return nil
		} else if len(pathSegments) == 1 {
			return val
		}

		return getDeepValue(val, strings.Join(pathSegments[1:], "."))
	case []interface{}:
		index, err := strconv.Atoi(pathSegments[0])
		if err != nil {
			return nil
		} else if index < 0 || index >= len(t) {
			return nil
		}

		val := t[index]
		if len(pathSegments) == 1 {
			return val
		}

		return getDeepValue(val, strings.Join(pathSegments[1:], "."))
	}

	return nil
}
