package workspace

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/driver/drivercreate"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// StartCmd holds the cmd flags
type StartCmd struct {
	*flags.GlobalFlags

	ID string
}

// NewStartCmd creates a new command
func NewStartCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &StartCmd{
		GlobalFlags: flags,
	}
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a workspace on the server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	startCmd.Flags().StringVar(&cmd.ID, "id", "", "The workspace id to start on the agent side")
	_ = startCmd.MarkFlagRequired("id")
	return startCmd
}

func (cmd *StartCmd) Run(ctx context.Context) error {
	// get workspace
	shouldExit, workspaceInfo, err := agent.ReadAgentWorkspaceInfo(cmd.AgentDir, cmd.Context, cmd.ID, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// create new docker client
	driver, err := drivercreate.NewDriver(workspaceInfo, log.Default)
	if err != nil {
		return err
	}

	// get container details
	containerDetails, err := driver.FindDevContainer(ctx, []string{
		config.DockerIDLabel + "=" + workspaceInfo.Workspace.ID,
	})
	if err != nil {
		return err
	} else if containerDetails == nil || containerDetails.State.Status != "running" {
		// start docker container
		_, err = StartContainer(workspaceInfo, log.Default)
		if err != nil {
			return errors.Wrap(err, "start container")
		}
	}

	return nil
}

func StartContainer(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (*config.Result, error) {
	log.Debugf("Starting DevPod container...")
	runner, err := createRunner(workspaceInfo, log)
	if err != nil {
		return nil, err
	}

	result, err := runner.Up(devcontainer.UpOptions{NoBuild: true})
	if err != nil {
		return result, err
	}
	log.Debugf("Successfully started DevPod container")
	return result, err
}
