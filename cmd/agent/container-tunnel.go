package agent

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/cli/cli"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

// ContainerTunnelCmd holds the ws-tunnel cmd flags
type ContainerTunnelCmd struct {
	ID    string
	Token string
}

// NewContainerTunnelCmd creates a new ws-tunnel command
func NewContainerTunnelCmd() *cobra.Command {
	cmd := &ContainerTunnelCmd{}
	containerTunnelCmd := &cobra.Command{
		Use:   "container-tunnel",
		Short: "Starts a new container ssh tunnel",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	containerTunnelCmd.Flags().StringVar(&cmd.ID, "id", "", "The id of the dev container")
	containerTunnelCmd.Flags().StringVar(&cmd.Token, "token", "", "The token to use for the ssh server")
	return containerTunnelCmd
}

// Run runs the command logic
func (cmd *ContainerTunnelCmd) Run(_ *cobra.Command, _ []string) error {
	// create new docker client
	dockerClient, err := docker.NewClient(context.Background(), log.Default)
	if err != nil {
		return err
	}

	// list all running containers
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("label", DockerIDLabel+"="+cmd.ID)),
	})
	if err != nil {
		return err
	} else if len(containers) == 0 {
		return fmt.Errorf("devcontainer is not running")
	}

	// get container
	container := &containers[0]

	// inject agent binary into container
	err = cmd.injectAgent(context.Background(), dockerClient, container, log.Default)
	if err != nil {
		return errors.Wrap(err, "inject agent")
	}

	// forward ssh into container
	err = cmd.forwardSSH(context.Background(), dockerClient, container.ID)
	if err != nil {
		return errors.Wrap(err, "forward ssh")
	}

	return nil
}

func (cmd *ContainerTunnelCmd) forwardSSH(ctx context.Context, client *docker.Client, container string) error {
	err := client.RawExec(ctx, container, "root", []string{agent.RemoteDevPodHelperLocation, "agent", "ssh-server", "--token", cmd.Token, "--stdio"}, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *ContainerTunnelCmd) injectAgent(ctx context.Context, client *docker.Client, container *types.Container, log log.Logger) error {
	// check if injecting is necessary
	injectNeeded, err := cmd.isInjectNeeded(ctx, client, container.ID)
	if err != nil {
		return err
	} else if !injectNeeded {
		return nil
	}

	// do the actual inject
	err = cmd.doInject(ctx, client, container.ID)
	if err != nil {
		return err
	}

	// set permissions
	err = cmd.setPermissions(ctx, client, container.ID)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *ContainerTunnelCmd) setPermissions(ctx context.Context, client *docker.Client, container string) error {
	buf := &bytes.Buffer{}
	err := client.RawExec(ctx, container, "root", []string{"sh", "-c", "chmod +x " + agent.RemoteDevPodHelperLocation}, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "set permissions: %s", buf.String())
	}

	return nil
}

func (cmd *ContainerTunnelCmd) doInject(ctx context.Context, client *docker.Client, container string) error {
	// start injecting the agent into the container
	currentBinary, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "find executable")
	}

	file, err := os.Open(currentBinary)
	if err != nil {
		return errors.Wrap(err, "open agent binary")
	}
	defer file.Close()

	// forward as stdin to command
	buf := &bytes.Buffer{}
	err = client.RawExec(ctx, container, "root", []string{"sh", "-c", fmt.Sprintf("cat > %s", agent.RemoteDevPodHelperLocation)}, file, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "inject: %s", buf.String())
	}

	return nil
}

func (cmd *ContainerTunnelCmd) isInjectNeeded(ctx context.Context, client *docker.Client, container string) (bool, error) {
	buf := &bytes.Buffer{}
	err := client.RawExec(ctx, container, "root", []string{agent.RemoteDevPodHelperLocation, "version"}, nil, buf, buf)
	if err != nil {
		if _, ok := err.(cli.StatusError); ok {
			return true, nil
		}

		return false, errors.Wrapf(err, "is inject needed: %s", buf.String())
	}

	return false, nil
}
