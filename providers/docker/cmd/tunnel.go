package cmd

import (
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

// TunnelCmd holds the cmd flags
type TunnelCmd struct{}

// NewTunnelCmd defines a command
func NewTunnelCmd() *cobra.Command {
	cmd := &TunnelCmd{}
	tunnelCmd := &cobra.Command{
		Use:   "tunnel",
		Short: "Tunnel a container",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), provider.FromEnvironment())
		},
	}

	return tunnelCmd
}

// Run runs the command logic
func (cmd *TunnelCmd) Run(ctx context.Context, workspace *provider.Workspace) error {
	runner := NewDockerProvider().newRunner(workspace, log.Default)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return err
	} else if containerDetails == nil {
		return nil
	}

	tok, err := token.GenerateWorkspaceToken(workspace.Context, workspace.ID)
	if err != nil {
		return errors.Wrap(err, "generate token")
	}

	err = runner.Docker.Tunnel(runner.AgentPath, runner.AgentDownloadURL, containerDetails.Id, tok, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	return nil
}
