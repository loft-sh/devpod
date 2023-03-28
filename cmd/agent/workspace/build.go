package workspace

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// BuildCmd holds the cmd flags
type BuildCmd struct {
	*flags.GlobalFlags

	ForceBuild    bool
	Repository    string
	WorkspaceInfo string
}

// NewBuildCmd creates a new command
func NewBuildCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &BuildCmd{
		GlobalFlags: flags,
	}
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Builds a devcontainer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	buildCmd.Flags().BoolVar(&cmd.ForceBuild, "force-build", false, "If true will force build the image")
	buildCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The repository to push to")
	buildCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = buildCmd.MarkFlagRequired("workspace-info")
	_ = buildCmd.MarkFlagRequired("repository")
	return buildCmd
}

// Run runs the command logic
func (cmd *BuildCmd) Run(ctx context.Context) error {
	// write workspace info
	shouldExit, workspaceInfo, err := agent.WriteWorkspaceInfoAndDeleteOld(cmd.WorkspaceInfo, deleteWorkspace, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// initialize the workspace
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	_, logger, credentialsDir, err := initWorkspace(cancelCtx, cancel, workspaceInfo, cmd.Debug, false)
	if err != nil {
		return err
	} else if credentialsDir != "" {
		defer func() {
			_ = os.RemoveAll(credentialsDir)
		}()
	}

	// build the image
	imageName, err := createRunner(workspaceInfo, logger).Build(devcontainer.BuildOptions{
		PushRepository: cmd.Repository,
		ForceRebuild:   cmd.ForceBuild,
	})
	if err != nil {
		logger.Errorf("Error building image: %v", err)
		return errors.Wrap(err, "build")
	}

	logger.Donef("Successfully build and pushed image %s", imageName)
	return nil
}

func deleteWorkspace(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	err := removeContainer(workspaceInfo, log)
	if err != nil {
		return errors.Wrap(err, "remove container")
	}

	_ = os.RemoveAll(filepath.Join(workspaceInfo.Folder, ".."))
	return nil
}
