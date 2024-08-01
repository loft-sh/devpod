package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/cmd/pro/provider/list"
	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/devpod/pkg/loft/kube"
	"github.com/loft-sh/devpod/pkg/loft/parameters"
	"github.com/loft-sh/devpod/pkg/loft/project"
	"github.com/loft-sh/devpod/pkg/loft/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/wait"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpCmd holds the cmd flags:
type UpCmd struct {
	*flags.GlobalFlags

	Log     log.Logger
	streams streams
}

type streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewUpCmd creates a new command
func NewUpCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
		streams: streams{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
	}
	c := &cobra.Command{
		Hidden: true,
		Use:    "up",
		Short:  "Runs up on a workspace",
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *UpCmd) Run(ctx context.Context) error {
	l := cmd.Log.ErrorStreamOnly()
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	info, err := GetWorkspaceInfoFromEnv()
	if err != nil {
		return err
	}
	workspace, err := FindWorkspace(ctx, baseClient, info.UID, info.ProjectName)
	if err != nil {
		return err
	}

	// create workspace if doesn't exist
	if workspace == nil {
		workspace, err = createWorkspace(ctx, baseClient, cmd.Log.ErrorStreamOnly())
		if err != nil {
			return fmt.Errorf("create workspace: %w", err)
		}

		return cmd.up(ctx, workspace, baseClient)
	}

	// update template if changed and user has permission
	if workspace.Spec.TemplateRef != nil && canUpdateWorkspace(ctx, workspace, baseClient, l) {
		oldWorkspace := workspace.DeepCopy()
		managementClient, err := baseClient.Management()
		if err != nil {
			return err
		}
		template := os.Getenv(loft.TemplateOptionEnv)
		if template == "" {
			return fmt.Errorf("%s is missing in environment", loft.TemplateOptionEnv)
		}
		version := os.Getenv(loft.TemplateVersionOptionEnv)
		if version == "latest" {
			version = ""
		}

		// set template and version
		workspace.Spec.TemplateRef = &storagev1.TemplateRef{
			Name:    template,
			Version: version,
		}

		// find parameters for template
		resolvedParameters, err := getParametersFromEnvironment(ctx, managementClient, info.ProjectName, template, version)
		if err != nil {
			return fmt.Errorf("resolve parameters: %w", err)
		}
		workspace.Spec.Parameters = resolvedParameters

		if workspaceChanged(workspace, oldWorkspace) {
			// only add `SyncOnce` if version didn't change
			if workspace.Spec.TemplateRef.Version == oldWorkspace.Spec.TemplateRef.Version {
				workspace.Spec.TemplateRef.SyncOnce = true
			}

			// update workspace resource
			workspace, err = managementClient.Loft().ManagementV1().
				DevPodWorkspaceInstances(project.ProjectNamespace(info.ProjectName)).
				Update(ctx, workspace, metav1.UpdateOptions{})
			if err != nil {
				return err
			}

			//  wait until status is updated
			err = wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
				workspace, err = managementClient.Loft().ManagementV1().
					DevPodWorkspaceInstances(project.ProjectNamespace(info.ProjectName)).
					Get(ctx, workspace.Name, metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				if !isReady(workspace) || !templateSynced(workspace) {
					l.Debugf("Workspace %s is in phase %s, waiting until its ready", workspace.Name, workspace.Status.Phase)
					return false, nil
				}

				l.Debugf("Workspace %s has been updated", workspace.Name)
				return true, nil
			})
			if err != nil {
				return fmt.Errorf("wait for instance to update: %w", err)
			}
		}
	}

	return cmd.up(ctx, workspace, baseClient)
}

func (cmd *UpCmd) up(ctx context.Context, workspace *managementv1.DevPodWorkspaceInstance, client client.Client) error {
	options := OptionsFromEnv(storagev1.DevPodFlagsUp)
	if options != nil && os.Getenv("DEBUG") == "true" {
		options.Add("debug", "true")
	}

	conn, err := DialWorkspace(client, workspace, "up", options, cmd.Log)
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, cmd.streams.Stdin, cmd.streams.Stdout, cmd.streams.Stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}

func createWorkspace(ctx context.Context, baseClient client.Client, log log.Logger) (*managementv1.DevPodWorkspaceInstance, error) {
	workspaceInfo, err := GetWorkspaceInfoFromEnv()
	if err != nil {
		return nil, err
	}

	// get template
	template := os.Getenv(loft.TemplateOptionEnv)
	if template == "" {
		return nil, fmt.Errorf("%s is missing in environment", loft.TemplateOptionEnv)
	}

	// create client
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	// get template version
	templateVersion := os.Getenv(loft.TemplateVersionOptionEnv)
	if templateVersion == "latest" {
		templateVersion = ""
	}

	// find parameters
	resolvedParameters, err := getParametersFromEnvironment(ctx, managementClient, workspaceInfo.ProjectName, template, templateVersion)
	if err != nil {
		return nil, fmt.Errorf("resolve parameters: %w", err)
	}

	// get workspace picture
	workspacePicture := os.Getenv("WORKSPACE_PICTURE")
	// get workspace source
	workspaceSource := os.Getenv("WORKSPACE_SOURCE")

	workspace := &managementv1.DevPodWorkspaceInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: encoding.SafeConcatNameMax([]string{workspaceInfo.ID}, 53) + "-",
			Namespace:    project.ProjectNamespace(workspaceInfo.ProjectName),
			Labels: map[string]string{
				storagev1.DevPodWorkspaceIDLabel:  workspaceInfo.ID,
				storagev1.DevPodWorkspaceUIDLabel: workspaceInfo.UID,
			},
			Annotations: map[string]string{
				storagev1.DevPodWorkspacePictureAnnotation: workspacePicture,
				storagev1.DevPodWorkspaceSourceAnnotation:  workspaceSource,
			},
		},
		Spec: managementv1.DevPodWorkspaceInstanceSpec{
			DevPodWorkspaceInstanceSpec: storagev1.DevPodWorkspaceInstanceSpec{
				DisplayName: workspaceInfo.ID,
				Parameters:  resolvedParameters,
				TemplateRef: &storagev1.TemplateRef{
					Name:    template,
					Version: templateVersion,
				},
			},
		},
	}

	// check if runner is defined
	runnerName := os.Getenv("LOFT_RUNNER")
	if runnerName != "" {
		workspace.Spec.RunnerRef.Runner = runnerName
	}

	// create instance
	workspace, err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(workspaceInfo.ProjectName)).Create(ctx, workspace, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	log.Infof("Created workspace %s", workspace.Name)

	// we need to wait until instance is scheduled
	err = wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
		workspace, err = managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(project.ProjectNamespace(workspaceInfo.ProjectName)).Get(ctx, workspace.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if !isReady(workspace) {
			log.Debugf("Workspace %s is in phase %s, waiting until its ready", workspace.Name, workspace.Status.Phase)
			return false, nil
		}

		if !isRunnerReady(workspace, storagev1.BuiltinRunnerName) {
			log.Debugf("Runner is not ready yet, waiting until its ready", workspace.Name, workspace.Status.Phase)
			return false, nil
		}

		log.Debugf("Workspace %s is ready", workspace.Name)
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("wait for instance to get ready: %w", err)
	}

	return workspace, nil
}

func getParametersFromEnvironment(ctx context.Context, kubeClient kube.Interface, projectName, templateName, templateVersion string) (string, error) {
	// are there any parameters in environment?
	environmentVariables := os.Environ()
	envMap := map[string]string{}
	for _, v := range environmentVariables {
		splitted := strings.SplitN(v, "=", 2)
		if len(splitted) != 2 {
			continue
		} else if !strings.HasPrefix(splitted[0], "TEMPLATE_OPTION_") {
			continue
		}

		envMap[splitted[0]] = splitted[1]
	}
	if len(envMap) == 0 {
		return "", nil
	}

	// find these in the template
	template, err := list.FindTemplate(ctx, kubeClient, projectName, templateName)
	if err != nil {
		return "", fmt.Errorf("find template: %w", err)
	}

	// find version
	var templateParameters []storagev1.AppParameter
	if len(template.Spec.Versions) > 0 {
		templateParameters, err = list.GetTemplateParameters(template, templateVersion)
		if err != nil {
			return "", err
		}
	} else {
		templateParameters = template.Spec.Parameters
	}

	// parse versions
	outMap := map[string]interface{}{}
	for _, parameter := range templateParameters {
		// check if its in environment
		val := envMap[list.VariableToEnvironmentVariable(parameter.Variable)]
		outVal, err := parameters.VerifyValue(val, parameter)
		if err != nil {
			return "", fmt.Errorf("validate parameter %s: %w", parameter.Variable, err)
		}

		outMap[parameter.Variable] = outVal
	}

	// convert to string
	out, err := yaml.Marshal(outMap)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func isReady(workspace *managementv1.DevPodWorkspaceInstance) bool {
	// Sleeping is considered ready in this context. The workspace will be woken up as soon as we connect to it
	if workspace.Status.Phase == storagev1.InstanceSleeping {
		return true
	}

	return workspace.Status.Phase == storagev1.InstanceReady
}

func isRunnerReady(workspace *managementv1.DevPodWorkspaceInstance, builtinRunnerName string) bool {
	if workspace.Spec.RunnerRef.Runner == "" {
		return true
	}

	if workspace.Spec.RunnerRef.Runner == builtinRunnerName {
		return true
	}

	return workspace.GetAnnotations() != nil &&
		workspace.GetAnnotations()[storagev1.DevPodWorkspaceRunnerEndpointAnnotation] != ""
}

func templateSynced(workspace *managementv1.DevPodWorkspaceInstance) bool {
	for _, condition := range workspace.Status.Conditions {
		if condition.Type == storagev1.InstanceTemplateResolved {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}

func workspaceChanged(newWorkspace, workspace *managementv1.DevPodWorkspaceInstance) bool {
	// compare template
	if !equality.Semantic.DeepEqual(workspace.Spec.TemplateRef, newWorkspace.Spec.TemplateRef) {
		return true
	}

	// compare parameters
	if !equality.Semantic.DeepEqual(workspace.Spec.Parameters, newWorkspace.Spec.Parameters) {
		return true
	}

	return false
}

func canUpdateWorkspace(ctx context.Context, workspace *managementv1.DevPodWorkspaceInstance, client client.Client, log log.Logger) bool {
	managementClient, err := client.Management()
	if err != nil {
		return false
	}

	review, err := managementClient.Loft().ManagementV1().SelfSubjectAccessReviews().Create(ctx,
		&managementv1.SelfSubjectAccessReview{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: managementv1.SelfSubjectAccessReviewSpec{
				SelfSubjectAccessReviewSpec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Verb:      "update",
						Group:     managementv1.SchemeGroupVersion.Group,
						Version:   managementv1.SchemeGroupVersion.Version,
						Resource:  "devpodworkspaceinstances",
						Namespace: workspace.Namespace,
						Name:      workspace.Name,
					},
				},
			},
		},
		metav1.CreateOptions{})

	if err != nil {
		log.Infof("self subject access review: %w", err)
		return false
	}
	if !review.Status.Allowed || review.Status.Denied {
		return false
	}

	return true
}
