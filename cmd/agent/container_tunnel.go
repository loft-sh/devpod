package agent

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/spf13/cobra"
	"os"
)

// ContainerTunnelCmd holds the ws-tunnel cmd flags
type ContainerTunnelCmd struct {
	Token         string
	WorkspaceInfo string
}

// NewContainerTunnelCmd creates a new command
func NewContainerTunnelCmd() *cobra.Command {
	cmd := &ContainerTunnelCmd{}
	containerTunnelCmd := &cobra.Command{
		Use:   "container-tunnel",
		Short: "Starts a new container ssh tunnel",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	containerTunnelCmd.Flags().StringVar(&cmd.Token, "token", "", "The token to use for the container ssh server")
	containerTunnelCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = containerTunnelCmd.MarkFlagRequired("token")
	_ = containerTunnelCmd.MarkFlagRequired("workspace-info")
	return containerTunnelCmd
}

// Run runs the command logic
func (cmd *ContainerTunnelCmd) Run(_ *cobra.Command, _ []string) error {
	// create new docker client
	dockerHelper := docker.DockerHelper{DockerCommand: "docker"}

	// get workspace info
	workspaceInfo, err := agent.WriteWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return err
	}

	// check if we need to become root
	shouldExit, err := agent.RerunAsRoot(workspaceInfo)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Rerun as root: %v", err)
		os.Exit(1)
	} else if shouldExit {
		os.Exit(0)
	}

	// get container details
	containerDetails, err := dockerHelper.FindDevContainer([]string{
		devcontainer.DockerIDLabel + "=" + workspaceInfo.Workspace.ID,
	})
	if err != nil {
		return err
	}

	// create tunnel into container.
	err = dockerHelper.Tunnel(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, containerDetails.Id, cmd.Token, os.Stdin, os.Stdout, os.Stderr, true)
	if err != nil {
		return err
	}

	return nil
}
