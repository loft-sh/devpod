package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// StatusCmd holds the cmd flags
type StatusCmd struct{}

// NewStatusCmd defines a command
func NewStatusCmd() *cobra.Command {
	cmd := &StatusCmd{}
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Status of a container",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), provider.FromEnvironment())
		},
	}

	return statusCmd
}

// Run runs the command logic
func (cmd *StatusCmd) Run(ctx context.Context, workspace *provider.Workspace) error {
	runner := NewDockerProvider().newRunner(workspace, log.Default)
	status, err := WorkspaceStatus(runner)
	if err != nil {
		return err
	}

	_, _ = os.Stdout.Write([]byte(status))
	return nil
}

func WorkspaceStatus(runner *devcontainer.Runner) (provider.Status, error) {
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return provider.StatusNotFound, err
	} else if containerDetails == nil {
		return provider.StatusNotFound, nil
	}

	status := strings.ToLower(containerDetails.State.Status)
	if status == "running" {
		return provider.StatusRunning, nil
	} else if status == "paused" {
		return provider.StatusBusy, nil
	} else if status == "exited" {
		return provider.StatusStopped, nil
	}

	return provider.StatusBusy, nil
}
