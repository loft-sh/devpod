package list

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/loft/client"
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

	projectName := os.Getenv("LOFT_PROJECT")
	if projectName == "" {
		return fmt.Errorf("LOFT_PROJECT environment variable is empty")
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
	templates := []string{}
	for _, template := range templateList.DevPodWorkspaceTemplates {
		templates = append(templates, template.Name)
	}
	sort.Strings(templates)

	runnerList, err := managementClient.Loft().ManagementV1().Projects().ListClusters(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("list runners: %w", err)
	} else if len(runnerList.Runners) == 0 {
		return fmt.Errorf("seems like there is no runner allowed in project %s, please make sure to at least have a single runner available", projectName)
	}

	// collect runners
	runners := []string{}
	for _, runner := range runnerList.Runners {
		runners = append(runners, runner.Name)
	}
	sort.Strings(runners)

	return printOptions(&OptionsFormat{
		Options: map[string]*Option{
			"LOFT_RUNNER": {
				DisplayName: "Runner",
				Description: "The DevPod.Pro runner to use for a new workspace.",
				Enum:        runners,
			},
			"LOFT_TEMPLATE": {
				DisplayName:       "Template",
				Description:       "The template to use for a new workspace.",
				Required:          true,
				Enum:              templates,
				Default:           templateList.DefaultDevPodWorkspaceTemplate,
				SubOptionsCommand: fmt.Sprintf("'%s' pro provider list templateoptions", executable),
				Mutable:           true,
			},
		},
	})
}
