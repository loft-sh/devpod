package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
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
	// get workspace
	workspaceInfo, err := agent.WriteWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return fmt.Errorf("error parsing workspace info: %v", err)
	}

	// check if we need to become root
	shouldExit, err := agent.RerunAsRoot(workspaceInfo)
	if err != nil {
		return err
	} else if shouldExit {
		return nil
	}

	// initialize the workspace
	tunnelClient, logger, err := initWorkspace(ctx, workspaceInfo, cmd.Debug, false)
	if err != nil {
		return err
	}

	// get docker credentials
	dir, err := configureDockerCredentials(ctx, workspaceInfo, tunnelClient, logger)
	if err != nil {
		logger.Errorf("Error retrieving docker credentials: %v", err)
	} else if dir != "" {
		defer os.RemoveAll(dir)
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
