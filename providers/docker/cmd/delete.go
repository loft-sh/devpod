package cmd

import (
	"bytes"
	"context"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the cmd flags
type DeleteCmd struct{}

// NewDeleteCmd defines a command
func NewDeleteCmd() *cobra.Command {
	cmd := &DeleteCmd{}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a container",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), provider.FromEnvironment())
		},
	}

	return deleteCmd
}

// Run runs the command logic
func (cmd *DeleteCmd) Run(ctx context.Context, workspace *provider.Workspace) error {
	dockerProvider := NewDockerProvider()
	runner := dockerProvider.newRunner(workspace, log.Default)
	containerDetails, err := runner.FindDevContainer()
	if err != nil {
		return err
	} else if containerDetails == nil {
		return nil
	}

	status, err := WorkspaceStatus(runner)
	if err != nil {
		return err
	} else if status == provider.StatusNotFound {
		return nil
	}

	// stop before removing
	if status == provider.StatusRunning {
		buf := &bytes.Buffer{}
		err = dockerProvider.docker.Run([]string{"stop", "-t", "5", containerDetails.Id}, nil, buf, buf)
		if err != nil {
			return errors.Wrapf(err, "stop container %s", buf.String())
		}
	}

	// remove container if stopped
	buf := &bytes.Buffer{}
	err = dockerProvider.docker.Run([]string{"rm", containerDetails.Id}, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "remove container %s", buf.String())
	}

	return nil
}
