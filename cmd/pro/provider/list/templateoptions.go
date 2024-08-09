package list

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/devpod/pkg/loft/kube"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TemplateOptionsCmd holds the cmd flags
type TemplateOptionsCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewTemplateOptionsCmd creates a new command
func NewTemplateOptionsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &TemplateOptionsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "templateoptions",
		Short: "Lists template options for the DevPod provider",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *TemplateOptionsCmd) Run(ctx context.Context) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	projectName := os.Getenv(loft.ProjectEnv)
	if projectName == "" {
		return fmt.Errorf("%s environment variable is empty", loft.ProjectEnv)
	}

	templateName := os.Getenv(loft.TemplateOptionEnv)
	if templateName == "" {
		return fmt.Errorf("%s environment variable is empty", loft.TemplateOptionEnv)
	}

	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// check template
	template, err := FindTemplate(ctx, managementClient, projectName, templateName)
	if err != nil {
		return err
	}

	// is template versioned?
	options := map[string]*types.Option{}
	if len(template.Spec.Versions) > 0 {
		versions := []types.OptionEnum{{Value: "latest", DisplayName: "Latest"}}
		for _, version := range template.Spec.Versions {
			versions = append(versions, types.OptionEnum{Value: version.Version})
		}

		options[loft.TemplateVersionOptionEnv] = &types.Option{
			DisplayName:       "Template Version",
			Description:       "The template version. If empty will use the latest version",
			Required:          true,
			Mutable:           true,
			Default:           "latest",
			Enum:              versions,
			SubOptionsCommand: fmt.Sprintf("'%s' pro provider list templateoptionsversion", executable),
		}
	} else {
		// parameters
		options = parametersToOptions(template.Spec.Parameters)
	}

	// print to stdout
	return printOptions(&OptionsFormat{Options: options})
}

var replaceRegEx = regexp.MustCompile("[^a-zA-Z0-9]+")

func parametersToOptions(parameters []storagev1.AppParameter) map[string]*types.Option {
	options := map[string]*types.Option{}
	for _, parameter := range parameters {
		optionName := VariableToEnvironmentVariable(parameter.Variable)
		displayName := parameter.Label
		if displayName == "" {
			displayName = optionName
		}
		paramOpts := []types.OptionEnum{}
		for _, opt := range parameter.Options {
			paramOpts = append(paramOpts, types.OptionEnum{Value: opt})
		}

		options[optionName] = &types.Option{
			DisplayName: displayName,
			Description: parameter.Description,
			Required:    parameter.Required,
			Enum:        paramOpts,
			Default:     parameter.DefaultValue,
			Mutable:     true,
			Type:        getOptionType(parameter),
			Password:    parameter.Type == "password",
		}
	}
	return options
}

var parameterToOptionTypeMap = map[string]string{
	"string":    "string",
	"multiline": "multiline",
	"number":    "number",
	"boolean":   "boolean",
}

func getOptionType(parameter storagev1.AppParameter) string {
	t, ok := parameterToOptionTypeMap[parameter.Type]
	if !ok {
		return ""
	}

	return t
}

func FindTemplate(ctx context.Context, managementClient kube.Interface, projectName, templateName string) (*managementv1.DevPodWorkspaceTemplate, error) {
	templateList, err := managementClient.Loft().ManagementV1().Projects().ListTemplates(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	} else if len(templateList.DevPodWorkspaceTemplates) == 0 {
		return nil, fmt.Errorf("seems like there is no DevPod template allowed in project %s, please make sure to at least have a single template available", projectName)
	}

	// find template
	var template *managementv1.DevPodWorkspaceTemplate
	for _, workspaceTemplate := range templateList.DevPodWorkspaceTemplates {
		if workspaceTemplate.Name == templateName {
			t := workspaceTemplate
			template = &t
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("couldn't find template %s", templateName)
	}

	return template, nil
}

func VariableToEnvironmentVariable(variable string) string {
	return "TEMPLATE_OPTION_" + strings.ToUpper(replaceRegEx.ReplaceAllString(variable, "_"))
}
