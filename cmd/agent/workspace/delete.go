package workspace

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/daemon"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// DeleteCmd holds the cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	Container bool
	Daemon    bool
	ID        string
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
	deleteCmd.Flags().StringVar(&cmd.ID, "id", "", "The workspace id to delete on the agent side")
	_ = deleteCmd.MarkFlagRequired("id")
	return deleteCmd
}

func (cmd *DeleteCmd) Run(ctx context.Context) error {
	// get workspace
	workspaceInfo, err := agent.ReadAgentWorkspaceInfo(cmd.Context, cmd.ID)
	if err != nil {
		return err
	}

	// check if we need to become root
	shouldExit, err := agent.RerunAsRoot(workspaceInfo)
	if err != nil {
		return errors.Wrap(err, "rerun as root")
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
		err = removeContainer(workspaceInfo, log.Default)
		if err != nil {
			return errors.Wrap(err, "remove container")
		}
	}

	// delete workspace folder
	_ = os.RemoveAll(filepath.Join(workspaceInfo.Folder, ".."))
	return nil
}

func removeContainer(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	log.Debugf("Removing DevPod container from server...")
	err := createRunner(workspaceInfo, log).Delete()
	if err != nil {
		return err
	}
	log.Debugf("Successfully removed DevPod container from server")

	return nil
}

func removeDaemon(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	if len(workspaceInfo.Workspace.Provider.Agent.Exec.Shutdown) == 0 {
		return nil
	}

	log.Debugf("Removing DevPod daemon from server...")
	err := daemon.RemoveDaemon()
	if err != nil {
		return errors.Wrap(err, "remove daemon")
	}
	log.Debugf("Successfully removed DevPod daemon from server")

	return nil
}
