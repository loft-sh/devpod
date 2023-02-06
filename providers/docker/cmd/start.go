package cmd

import (
	"bytes"
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StartCmd holds the cmd flags
type StartCmd struct{}

// NewStartCmd defines a command
func NewStartCmd() *cobra.Command {
	cmd := &StartCmd{}
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a container",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), provider.FromEnvironment())
		},
	}

	return startCmd
}

// Run runs the command logic
func (cmd *StartCmd) Run(ctx context.Context, workspace *provider.Workspace) error {
	dockerProvider := NewDockerProvider()
	runner := dockerProvider.newRunner(workspace, log.Default)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return err
	} else if containerDetails == nil {
		return nil
	}

	buf := &bytes.Buffer{}
	err = dockerProvider.docker.Run([]string{"start", containerDetails.Id}, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "start container %s", buf.String())
	}

	return nil
}
