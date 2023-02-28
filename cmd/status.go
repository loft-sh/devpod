package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
)

// StatusCmd holds the cmd flags
type StatusCmd struct {
	*flags.GlobalFlags

	Output          string
	ContainerStatus bool
}

// NewStatusCmd creates a new command
func NewStatusCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StatusCmd{
		GlobalFlags: flags,
	}
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Shows the status of a workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			client, err := workspace2.GetWorkspace(ctx, devPodConfig, nil, args, log.Default)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, client)
		},
	}

	statusCmd.Flags().BoolVar(&cmd.ContainerStatus, "container-status", true, "If enabled shows the workspace container status as well")
	statusCmd.Flags().StringVar(&cmd.Output, "output", "plain", "Status shows the workspace status")
	return statusCmd
}

// Run runs the command logic
func (cmd *StatusCmd) Run(ctx context.Context, client client2.WorkspaceClient) error {
	// get instance status
	instanceStatus, err := client.Status(ctx, client2.StatusOptions{ContainerStatus: cmd.ContainerStatus})
	if err != nil {
		return err
	}

	if cmd.Output == "plain" {
		if instanceStatus == client2.StatusStopped {
			log.Default.Infof("Workspace '%s' is '%s', you can start it via 'devpod up %s'", client.Workspace(), instanceStatus, client.Workspace())
		} else if instanceStatus == client2.StatusBusy {
			log.Default.Infof("Workspace '%s' is '%s', which means its currently unaccessible. This is usually resolved by waiting a couple of minutes", client.Workspace(), instanceStatus)
		} else if instanceStatus == client2.StatusNotFound {
			log.Default.Infof("Workspace '%s' is '%s', you can create it via 'devpod up %s'", client.Workspace(), instanceStatus, client.Workspace())
		} else {
			log.Default.Infof("Workspace '%s' is '%s'", client.Workspace(), instanceStatus)
		}
	} else if cmd.Output == "json" {
		out, err := json.Marshal(struct {
			ID       string `json:"id,omitempty"`
			Context  string `json:"context,omitempty"`
			Provider string `json:"provider,omitempty"`
			State    string `json:"state,omitempty"`
		}{
			ID:       client.Workspace(),
			Context:  client.Context(),
			Provider: client.Provider(),
			State:    string(instanceStatus),
		})
		if err != nil {
			return err
		}

		fmt.Print(string(out))
	} else {
		return fmt.Errorf("unexpected output format, choose either json or plain. Got %s", cmd.Output)
	}

	return nil
}
