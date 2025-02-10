package list

import (
	"context"
	"encoding/json"
	"fmt"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform/client"
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
		Short: "Lists projects for the provider",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *ProjectsCmd) Run(ctx context.Context) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	projectList, err := Projects(ctx, baseClient)
	if err != nil {
		return err
	}

	out, err := json.Marshal(projectList.Items)
	if err != nil {
		return err
	}

	fmt.Println(string(out))

	return nil
}

func Projects(ctx context.Context, client client.Client) (*managementv1.ProjectList, error) {
	managementClient, err := client.Management()
	if err != nil {
		return nil, err
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return projectList, fmt.Errorf("list projects: %w", err)
	} else if len(projectList.Items) == 0 {
		return projectList, fmt.Errorf("you don't have access to any projects, please make sure you have at least access to 1 project")
	}

	return projectList, nil
}
