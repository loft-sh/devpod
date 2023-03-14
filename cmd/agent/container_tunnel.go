package agent

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/agent/workspace"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ContainerTunnelCmd holds the ws-tunnel cmd flags
type ContainerTunnelCmd struct {
	Token         string
	WorkspaceInfo string
	User          string

	TrackActivity  bool
	StartContainer bool
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

	containerTunnelCmd.Flags().BoolVar(&cmd.TrackActivity, "track-activity", false, "If true, tracks the activity in the container")
	containerTunnelCmd.Flags().StringVar(&cmd.User, "user", "", "The user to create the tunnel with")
	containerTunnelCmd.Flags().BoolVar(&cmd.StartContainer, "start-container", false, "If true, will try to start the container")
	containerTunnelCmd.Flags().StringVar(&cmd.Token, "token", "", "The token to use for the container ssh server")
	containerTunnelCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = containerTunnelCmd.MarkFlagRequired("token")
	_ = containerTunnelCmd.MarkFlagRequired("workspace-info")
	return containerTunnelCmd
}

// Run runs the command logic
func (cmd *ContainerTunnelCmd) Run(_ *cobra.Command, _ []string) error {
	// write workspace info
	shouldExit, workspaceInfo, err := agent.WriteWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// wait until devcontainer is started
	containerId := ""
	if cmd.StartContainer {
		containerId, err = startDevContainer(workspaceInfo)
	} else {
		containerId, err = waitForDevContainer(workspaceInfo)
	}
	if err != nil {
		return err
	}

	// handle SIGHUP
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	go func() {
		<-sigs
		os.Exit(0)
	}()

	// create tunnel into container.
	err = agent.Tunnel(
		context.TODO(),
		docker.NewDockerHelper(),
		agent.RemoteDevPodHelperLocation,
		agent.DefaultAgentDownloadURL,
		containerId,
		cmd.Token,
		cmd.User,
		os.Stdin,
		os.Stdout,
		os.Stderr,
		cmd.TrackActivity,
		log.Default.ErrorStreamOnly(),
	)
	if err != nil {
		return err
	}

	return nil
}

func waitForDevContainer(workspaceInfo *provider2.AgentWorkspaceInfo) (string, error) {
	dockerHelper := docker.NewDockerHelper()
	now := time.Now()
	for time.Since(now) < time.Minute*2 {
		containerDetails, err := dockerHelper.FindDevContainer([]string{
			devcontainer.DockerIDLabel + "=" + workspaceInfo.Workspace.ID,
		})
		if err != nil {
			return "", err
		} else if containerDetails == nil || containerDetails.State.Status != "running" {
			time.Sleep(time.Second)
			continue
		}

		return containerDetails.Id, nil
	}

	return "", fmt.Errorf("timed out waiting for devcontainer to come up")
}

func startDevContainer(workspaceInfo *provider2.AgentWorkspaceInfo) (string, error) {
	dockerHelper := docker.NewDockerHelper()
	containerDetails, err := dockerHelper.FindDevContainer([]string{
		devcontainer.DockerIDLabel + "=" + workspaceInfo.Workspace.ID,
	})
	if err != nil {
		return "", err
	} else if containerDetails == nil || containerDetails.State.Status != "running" {
		// start container
		result, err := workspace.StartContainer(workspaceInfo, log.Default.ErrorStreamOnly())
		if err != nil {
			return "", err
		}

		return result.ContainerDetails.Id, nil
	}

	return containerDetails.Id, nil
}
