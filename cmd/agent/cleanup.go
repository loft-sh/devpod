package agent

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/daemon"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// CleanupCmd holds the cmd flags
type CleanupCmd struct {
	flags.GlobalFlags

	Container     bool
	Daemon        bool
	WorkspaceInfo string
}

// NewCleanupCmd creates a new command
func NewCleanupCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &CleanupCmd{
		GlobalFlags: *flags,
	}
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleans up a workspace on the remote server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	cleanupCmd.Flags().BoolVar(&cmd.Container, "container", true, "If enabled, cleans up the DevPod container")
	cleanupCmd.Flags().BoolVar(&cmd.Daemon, "daemon", true, "If enabled, cleans up the DevPod daemon")
	cleanupCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = cleanupCmd.MarkFlagRequired("workspace-info")
	return cleanupCmd
}

func (cmd *CleanupCmd) Run(ctx context.Context) error {
	// get workspace
	workspaceInfo, err := getWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return err
	}

	// check if we need to become root
	shouldExit, err := rerunAsRoot(workspaceInfo)
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
	err := devcontainer.NewRunner(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, workspaceInfo.Folder, workspaceInfo.Workspace.ID, log).Delete()
	if err != nil {
		return err
	}
	log.Debugf("Successfully removed DevPod container from server")

	return nil
}

func removeDaemon(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	if workspaceInfo.AgentConfig == nil || len(workspaceInfo.AgentConfig.Exec.Shutdown) == 0 {
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
