package list

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TemplatesCmd holds the cmd flags
type TemplatesCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewTemplatesCmd creates a new command
func NewTemplatesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &TemplatesCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "templates",
		Short: "Lists templates for the DevPod provider",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *TemplatesCmd) Run(ctx context.Context) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	projectName := os.Getenv(loft.ProjectEnv)
	if projectName == "" {
		return fmt.Errorf("%s environment variable is empty", loft.ProjectEnv)
	}

	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	templateList, err := managementClient.Loft().ManagementV1().Projects().ListTemplates(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("list templates: %w", err)
	} else if len(templateList.DevPodWorkspaceTemplates) == 0 {
		return fmt.Errorf("seems like there is no DevPod template allowed in project %s, please make sure to at least have a single template available", projectName)
	}

	// collect templates
	templates := []types.OptionEnum{}
	for _, template := range templateList.DevPodWorkspaceTemplates {
		templates = append(templates, types.OptionEnum{
			Value:       template.Name,
			DisplayName: loft.DisplayName(template.Name, template.Spec.DisplayName),
		})
	}
	slices.SortFunc(templates, func(a types.OptionEnum, b types.OptionEnum) int {
		return cmp.Compare(a.Value, b.Value)
	})

	runnerList, err := managementClient.Loft().ManagementV1().Projects().ListClusters(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("list runners: %w", err)
	} else if len(runnerList.Runners) == 0 {
		return fmt.Errorf("seems like there is no runner allowed in project %s, please make sure to at least have a single runner available", projectName)
	}

	// collect runners
	runners := []types.OptionEnum{}
	for _, runner := range runnerList.Runners {
		runners = append(runners, types.OptionEnum{
			Value:       runner.Name,
			DisplayName: loft.DisplayName(runner.Name, runner.Spec.DisplayName),
		})
	}
	slices.SortFunc(runners, func(a types.OptionEnum, b types.OptionEnum) int {
		return cmp.Compare(a.Value, b.Value)
	})

	//collect environments
	environmentsList, err := managementClient.Loft().ManagementV1().DevPodEnvironmentTemplates().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list environments: %w", err)
	}

	environments := []types.OptionEnum{}
	for _, env := range environmentsList.Items {
		environments = append(environments, types.OptionEnum{
			Value:       env.Name,
			DisplayName: loft.DisplayName(env.Name, env.Spec.DisplayName),
		})
	}

	options := map[string]*types.Option{
		loft.RunnerEnv: {
			DisplayName: "Runner",
			Description: "The DevPod Pro runner to use for a new workspace.",
			Enum:        runners,
			Required:    true,
			Mutable:     false,
		},
		loft.TemplateOptionEnv: {
			DisplayName:       "Template",
			Description:       "The template to use for a new workspace.",
			Required:          true,
			Enum:              templates,
			Default:           templateList.DefaultDevPodWorkspaceTemplate,
			SubOptionsCommand: fmt.Sprintf("'%s' pro provider list templateoptions", executable),
			Mutable:           true,
		},
	}

	if len(environments) > 0 {
		options[loft.EnvironmentTemplateOptionEnv] = &types.Option{
			DisplayName: "Environment Template",
			Description: "The template to use for creating environment",
			Enum:        environments,
			Required:    true,
			Mutable:     false,
		}
	}

	return printOptions(&OptionsFormat{
		Options: options,
	})
}
