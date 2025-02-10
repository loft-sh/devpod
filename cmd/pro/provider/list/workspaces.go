package list

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/platform/labels"
	"github.com/loft-sh/devpod/pkg/platform/project"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkspacesCmd holds the cmd flags
type WorkspacesCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewWorkspacesCmd creates a new command
func NewWorkspacesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &WorkspacesCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "workspaces",
		Short: "Lists workspaces for the provider",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *WorkspacesCmd) Run(ctx context.Context) error {
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

	filterByOwner := os.Getenv(provider.LOFT_FILTER_BY_OWNER) == "true"
	workspaces := []*managementv1.DevPodWorkspaceInstance{}
	for _, p := range projectList.Items {
		ns := project.ProjectNamespace(p.GetName())
		workspaceList, err := managementClient.Loft().ManagementV1().DevPodWorkspaceInstances(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			cmd.log.Info("list workspaces in project \"%s\": %w", p.GetName(), err)
			continue
		}

		for _, instance := range workspaceList.Items {
			instance := &instance
			if filterByOwner && !platform.IsOwner(baseClient.Self(), instance.GetOwner()) {
				continue
			}

			if instance.GetLabels() == nil {
				instance.Labels = map[string]string{}
			}
			instance.Labels[labels.ProjectLabel] = p.GetName()

			workspaces = append(workspaces, instance)
		}
	}

	wBytes, err := json.Marshal(workspaces)
	if err != nil {
		return fmt.Errorf("marshal workspaces: %w", err)
	}
	fmt.Println(string(wBytes))

	return nil
}
