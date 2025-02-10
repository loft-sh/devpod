package machine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
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
		Use:   "status [name]",
		Short: "Retrieves the status of an existing machine",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	statusCmd.Flags().StringVar(&cmd.Output, "output", "plain", "Status shows the machine status")
	return statusCmd
}

// Run runs the command logic
func (cmd *StatusCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	machineClient, err := workspace.GetMachine(devPodConfig, args, log.Default)
	if err != nil {
		return err
	}

	// get status
	machineStatus, err := machineClient.Status(ctx, client.StatusOptions{})
	if err != nil {
		return err
	}

	if cmd.Output == "plain" {
		if machineStatus == client.StatusStopped {
			log.Default.Infof("Machine '%s' is '%s', you can start it via 'devpod machine start %s'", machineClient.Machine(), machineStatus, machineClient.Machine())
		} else if machineStatus == client.StatusBusy {
			log.Default.Infof("Machine '%s' is '%s', which means its currently unaccessible. This is usually resolved by waiting a couple of minutes", machineClient.Machine(), machineStatus)
		} else if machineStatus == client.StatusNotFound {
			log.Default.Infof("Machine '%s' is '%s'", machineClient.Machine(), machineStatus)
		} else {
			log.Default.Infof("Machine '%s' is '%s'", machineClient.Machine(), machineStatus)
		}
	} else if cmd.Output == "json" {
		out, err := json.Marshal(struct {
			ID       string `json:"id,omitempty"`
			Context  string `json:"context,omitempty"`
			Provider string `json:"provider,omitempty"`
			State    string `json:"state,omitempty"`
		}{
			ID:       machineClient.Machine(),
			Context:  machineClient.Context(),
			Provider: machineClient.Provider(),
			State:    string(machineStatus),
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
