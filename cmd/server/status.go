package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
)

// StatusCmd holds the configuration
type StatusCmd struct {
	*flags.GlobalFlags

	Output string
}

// NewStatusCmd creates a new destroy command
func NewStatusCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StatusCmd{
		GlobalFlags: flags,
	}
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Retrieves the status of an existing server",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	statusCmd.Flags().StringVar(&cmd.Output, "output", "plain", "Status shows the server status")
	return statusCmd
}

// Run runs the command logic
func (cmd *StatusCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context)
	if err != nil {
		return err
	}

	serverClient, err := workspace.GetServer(ctx, devPodConfig, args, log.Default)
	if err != nil {
		return err
	}

	// get status
	serverStatus, err := serverClient.Status(ctx, client.StatusOptions{})
	if err != nil {
		return err
	}

	if cmd.Output == "plain" {
		if serverStatus == client.StatusStopped {
			log.Default.Infof("Server '%s' is '%s', you can start it via 'devpod server start %s'", serverClient.Server(), serverStatus, serverClient.Server())
		} else if serverStatus == client.StatusBusy {
			log.Default.Infof("Server '%s' is '%s', which means its currently unaccessible. This is usually resolved by waiting a couple of minutes", serverClient.Server(), serverStatus)
		} else if serverStatus == client.StatusNotFound {
			log.Default.Infof("Server '%s' is '%s'", serverClient.Server(), serverStatus)
		} else {
			log.Default.Infof("Server '%s' is '%s'", serverClient.Server(), serverStatus)
		}
	} else if cmd.Output == "json" {
		out, err := json.Marshal(struct {
			ID       string `json:"id,omitempty"`
			Context  string `json:"context,omitempty"`
			Provider string `json:"provider,omitempty"`
			State    string `json:"state,omitempty"`
		}{
			ID:       serverClient.Server(),
			Context:  serverClient.Context(),
			Provider: serverClient.Provider(),
			State:    string(serverStatus),
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
