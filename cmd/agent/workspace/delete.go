package workspace

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	agentdaemon "github.com/loft-sh/devpod/pkg/daemon/agent"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	Container bool
	Daemon    bool

	WorkspaceInfo string
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: flags,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Cleans up a workspace on the remote server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	deleteCmd.Flags().BoolVar(&cmd.Container, "container", true, "If enabled, cleans up the DevPod container")
	deleteCmd.Flags().BoolVar(&cmd.Daemon, "daemon", false, "If enabled, cleans up the DevPod daemon")

	deleteCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = deleteCmd.MarkFlagRequired("workspace-info")
	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context) error {
	// get workspace
	shouldExit, workspaceInfo, err := agent.WorkspaceInfo(cmd.WorkspaceInfo, log.Default.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error parsing workspace info: %w", err)
	} else if shouldExit {
		return nil
	}

	// remove daemon
	if cmd.Daemon {
		err = removeDaemon(workspaceInfo, log.Default)
		if err != nil {
			return errors.Wrap(err, "remove daemon")
		}
	}

	// cleanup docker container
	if cmd.Container {
		err = removeContainer(ctx, workspaceInfo, log.Default)
		if err != nil {
			return errors.Wrap(err, "remove container")
		}
	}

	// delete workspace folder
	_ = os.RemoveAll(workspaceInfo.Origin)
	return nil
}

func removeContainer(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	log.Debugf("Removing DevPod container from server...")
	runner, err := CreateRunner(workspaceInfo, log)
	if err != nil {
		return err
	}

	if workspaceInfo.Workspace.Source.Container != "" {
		log.Infof("Skipping container deletion, since it was not created by DevPod")
	} else {
		err = runner.Delete(ctx)
		if err != nil {
			return err
		}
		log.Debugf("Successfully removed DevPod container from server")
	}

	return nil
}

func removeDaemon(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	if len(workspaceInfo.Agent.Exec.Shutdown) == 0 {
		return nil
	}

	log.Debugf("Removing DevPod daemon from server...")
	err := agentdaemon.RemoveDaemon()
	if err != nil {
		return errors.Wrap(err, "remove daemon")
	}
	log.Debugf("Successfully removed DevPod daemon from server")

	return nil
}
