package agent

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/loft-sh/devpod/cmd/agent/workspace"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/devcontainer/setup"
	"github.com/loft-sh/devpod/pkg/encoding"
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
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.TODO(), log.Default.ErrorStreamOnly())
		},
	}

	containerTunnelCmd.Flags().StringVar(&cmd.User, "user", "", "The user to create the tunnel with")
	containerTunnelCmd.Flags().StringVar(&cmd.Token, "token", "", "The token to use for the container ssh server")
	containerTunnelCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = containerTunnelCmd.MarkFlagRequired("token")
	_ = containerTunnelCmd.MarkFlagRequired("workspace-info")
	return containerTunnelCmd
}

// Run runs the command logic
func (cmd *ContainerTunnelCmd) Run(ctx context.Context, log log.Logger) error {
	// write workspace info
	shouldExit, workspaceInfo, err := agent.WriteWorkspaceInfo(cmd.WorkspaceInfo, log)
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// create runner
	runner, err := workspace.CreateRunner(workspaceInfo, log)
	if err != nil {
		return err
	}

	// wait until devcontainer is started
	containerID, err := startDevContainer(ctx, workspaceInfo, runner, log)
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
		ctx,
		func(ctx context.Context, user string, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return runner.CommandDevContainer(ctx, containerID, user, command, stdin, stdout, stderr)
		},
		cmd.Token,
		cmd.User,
		os.Stdin,
		os.Stdout,
		os.Stderr,
		log,
	)
	if err != nil {
		return err
	}

	return nil
}

func startDevContainer(ctx context.Context, workspaceConfig *provider2.AgentWorkspaceInfo, runner *devcontainer.Runner, log log.Logger) (string, error) {
	containerDetails, err := runner.FindDevContainer(ctx)
	if err != nil {
		return "", err
	}

	// start container if necessary
	if containerDetails == nil || containerDetails.State.Status != "running" {
		// start container
		result, err := workspace.StartContainer(ctx, runner, log)
		if err != nil {
			return "", err
		}

		return result.ContainerDetails.ID, nil
	} else if encoding.IsLegacyUID(workspaceConfig.Workspace.UID) {
		// make sure workspace result is in devcontainer
		buf := &bytes.Buffer{}
		err = runner.CommandDevContainer(ctx, containerDetails.ID, "root", "cat "+setup.ResultLocation, nil, buf, buf)
		if err != nil {
			// start container
			result, err := workspace.StartContainer(ctx, runner, log)
			if err != nil {
				return "", err
			}

			return result.ContainerDetails.ID, nil
		}
	}

	return containerDetails.ID, nil
}
