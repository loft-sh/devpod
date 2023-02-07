package agent

import (
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/docker"
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
	_ = containerTunnelCmd.MarkFlagRequired("id")
	_ = containerTunnelCmd.MarkFlagRequired("token")
	return containerTunnelCmd
}

// Run runs the command logic
func (cmd *ContainerTunnelCmd) Run(_ *cobra.Command, _ []string) error {
	// create new docker client
	dockerHelper := docker.DockerHelper{DockerCommand: "docker"}

	// get container details
	containerDetails, err := dockerHelper.FindDevContainer([]string{
		devcontainer.DockerIDLabel + "=" + cmd.ID,
	})
	if err != nil {
		return err
	}

	// create tunnel
	err = dockerHelper.Tunnel(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, containerDetails.Id, cmd.Token, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	return nil
}
