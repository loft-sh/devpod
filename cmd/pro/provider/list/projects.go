package list

import (
	"cmp"
	"context"
	"encoding/json"
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

// ProjectsCmd holds the cmd flags
type ProjectsCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewProjectsCmd creates a new command
func NewProjectsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ProjectsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "projects",
		Short: "Lists projects for the DevPod provider",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *ProjectsCmd) Run(ctx context.Context) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	} else if len(projectList.Items) == 0 {
		return fmt.Errorf("you don't have access to any projects within DevPod Pro, please make sure you have at least access to 1 project")
	}

	enum := []types.OptionEnum{}
	for _, project := range projectList.Items {
		// Filter out projects that don't have allowed runners
		if project.Spec.AllowedRunners == nil || len(project.Spec.AllowedRunners) == 0 {
			continue
		}
		enum = append(enum, types.OptionEnum{
			Value:       project.Name,
			DisplayName: loft.DisplayName(project.Name, project.Spec.DisplayName),
		})
	}
	slices.SortFunc(enum, func(a types.OptionEnum, b types.OptionEnum) int {
		return cmp.Compare(a.Value, b.Value)
	})

	return printOptions(&OptionsFormat{
		Options: map[string]*types.Option{
			loft.ProjectEnv: {
				DisplayName:       "Project",
				Description:       "The DevPod Pro project to use to create a new workspace in.",
				Required:          true,
				Enum:              enum,
				Default:           enum[0].Value,
				SubOptionsCommand: fmt.Sprintf("'%s' pro provider list templates", executable),
			},
		},
	})
}

func printOptions(options *OptionsFormat) error {
	out, err := json.Marshal(options)
	if err != nil {
		return err
	}

	fmt.Print(string(out))
	return nil
}

type OptionsFormat struct {
	// Options holds the provider options
	Options map[string]*types.Option `json:"options,omitempty"`
}
