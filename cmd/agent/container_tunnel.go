package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/loft-sh/devpod/cmd/agent/workspace"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver"
	"github.com/loft-sh/devpod/pkg/driver/drivercreate"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/spf13/cobra"
)

// ContainerTunnelCmd holds the ws-tunnel cmd flags
type ContainerTunnelCmd struct {
	*flags.GlobalFlags

	Token         string
	WorkspaceInfo string
	User          string

	TrackActivity  bool
	StartContainer bool
}

// NewContainerTunnelCmd creates a new command
func NewContainerTunnelCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ContainerTunnelCmd{
		GlobalFlags: flags,
	}
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
	shouldExit, workspaceInfo, err := agent.WriteWorkspaceInfo(cmd.WorkspaceInfo, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// create driver
	driver, err := drivercreate.NewDriver(workspaceInfo, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	}

	// wait until devcontainer is started
	containerID := ""
	if cmd.StartContainer {
		containerID, err = startDevContainer(workspaceInfo, driver)
	} else {
		containerID, err = waitForDevContainer(workspaceInfo, driver)
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
		driver,
		containerID,
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

func waitForDevContainer(workspaceInfo *provider2.AgentWorkspaceInfo, driver driver.Driver) (string, error) {
	now := time.Now()
	for time.Since(now) < time.Minute*2 {
		containerDetails, err := driver.FindDevContainer(context.TODO(), []string{
			config.DockerIDLabel + "=" + workspaceInfo.Workspace.ID,
		})
		if err != nil {
			return "", err
		} else if containerDetails == nil || containerDetails.State.Status != "running" {
			time.Sleep(time.Second)
			continue
		}

		return containerDetails.ID, nil
	}

	return "", fmt.Errorf("timed out waiting for devcontainer to come up")
}

func startDevContainer(workspaceInfo *provider2.AgentWorkspaceInfo, driver driver.Driver) (string, error) {
	containerDetails, err := driver.FindDevContainer(context.TODO(), []string{
		config.DockerIDLabel + "=" + workspaceInfo.Workspace.ID,
	})
	if err != nil {
		return "", err
	} else if containerDetails == nil || containerDetails.State.Status != "running" {
		// start container
		result, err := workspace.StartContainer(workspaceInfo, log.Default.ErrorStreamOnly())
		if err != nil {
			return "", err
		}

		return result.ContainerDetails.ID, nil
	}

	return containerDetails.ID, nil
}
