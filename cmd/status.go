package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StatusCmd holds the cmd flags
type StatusCmd struct {
	*flags.GlobalFlags
	client2.StatusOptions

	Output  string
	Timeout string
}

// NewStatusCmd creates a new command
func NewStatusCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StatusCmd{
		GlobalFlags: flags,
	}
	statusCmd := &cobra.Command{
		Use:   "status [flags] [workspace-path|workspace-name]",
		Short: "Shows the status of a workspace",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			_, err := clientimplementation.DecodeOptionsFromEnv(clientimplementation.DevPodFlagsStatus, &cmd.StatusOptions)
			if err != nil {
				return fmt.Errorf("decode up options: %w", err)
			}

			ctx := cobraCmd.Context()
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			logger := log.Default.ErrorStreamOnly()
			client, err := workspace2.Get(ctx, devPodConfig, args, false, logger)
			if err != nil {
				return err
			}

			return cmd.Run(ctx, client, logger)
		},
	}

	statusCmd.Flags().BoolVar(&cmd.ContainerStatus, "container-status", true, "If enabled shows the workspace container status as well")
	statusCmd.Flags().StringVar(&cmd.Output, "output", "plain", "Status shows the workspace status")
	statusCmd.Flags().StringVar(&cmd.Timeout, "timeout", "30s", "The timeout to wait until the status can be retrieved")
	return statusCmd
}

// Run runs the command logic
func (cmd *StatusCmd) Run(ctx context.Context, client client2.BaseWorkspaceClient, log log.Logger) error {
	// parse timeout
	if cmd.Timeout != "" {
		duration, err := time.ParseDuration(cmd.Timeout)
		if err != nil {
			return errors.Wrap(err, "parse --timeout")
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	// get instance status
	instanceStatus, err := client.Status(ctx, cmd.StatusOptions)
	if err != nil {
		return err
	}

	if cmd.Output == "plain" {
		if instanceStatus == client2.StatusStopped {
			log.Infof("Workspace '%s' is '%s', you can start it via 'devpod up %s'", client.Workspace(), instanceStatus, client.Workspace())
		} else if instanceStatus == client2.StatusBusy {
			log.Infof("Workspace '%s' is '%s', which means its currently unaccessible. This is usually resolved by waiting a couple of minutes", client.Workspace(), instanceStatus)
		} else if instanceStatus == client2.StatusNotFound {
			log.Infof("Workspace '%s' is '%s', you can create it via 'devpod up %s'", client.Workspace(), instanceStatus, client.Workspace())
		} else {
			log.Infof("Workspace '%s' is '%s'", client.Workspace(), instanceStatus)
		}
	} else if cmd.Output == "json" {
		out, err := json.Marshal(&client2.WorkspaceStatus{
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
