package cmd

import (
	"bytes"
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StopCmd holds the cmd flags
type StopCmd struct{}

// NewStopCmd defines a command
func NewStopCmd() *cobra.Command {
	cmd := &StopCmd{}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a container",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), provider.FromEnvironment())
		},
	}

	return stopCmd
}

// Run runs the command logic
func (cmd *StopCmd) Run(ctx context.Context, workspace *provider.Workspace) error {
	dockerProvider := NewDockerProvider()
	runner := dockerProvider.newRunner(workspace, log.Default)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return err
	} else if containerDetails == nil {
		return nil
	}

	buf := &bytes.Buffer{}
	err = dockerProvider.docker.Run([]string{"stop", containerDetails.Id}, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "stop container %s", buf.String())
	}

	return nil
}
