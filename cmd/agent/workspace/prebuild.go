package workspace

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// PrebuildCmd holds the cmd flags
type PrebuildCmd struct {
	*flags.GlobalFlags

	ForceBuild    bool
	Repository    string
	WorkspaceInfo string
}

// NewPrebuildCmd creates a new command
func NewPrebuildCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &PrebuildCmd{
		GlobalFlags: flags,
	}
	prebuildCmd := &cobra.Command{
		Use:   "prebuild",
		Short: "Prebuilds a devcontainer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	prebuildCmd.Flags().BoolVar(&cmd.ForceBuild, "force-build", false, "If true will force build the image")
	prebuildCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The repository to push to")
	prebuildCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = prebuildCmd.MarkFlagRequired("workspace-info")
	_ = prebuildCmd.MarkFlagRequired("repository")
	return prebuildCmd
}

// Run runs the command logic
func (cmd *PrebuildCmd) Run(ctx context.Context) error {
	// get workspace
	workspaceInfo, err := agent.WriteWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return fmt.Errorf("error parsing workspace info: %v", err)
	}

	// check if we need to become root
	shouldExit, err := agent.RerunAsRoot(workspaceInfo)
	if err != nil {
		return fmt.Errorf("rerun as root: %v", err)
	} else if shouldExit {
		return nil
	}

	// initialize the workspace
	_, logger, err := initWorkspace(ctx, workspaceInfo, cmd.Debug)
	if err != nil {
		return err
	}

	// prebuild the image
	imageName, err := createRunner(workspaceInfo, logger).Prebuild(devcontainer.PrebuildOptions{
		PushRepository: cmd.Repository,
		ForceRebuild:   cmd.ForceBuild,
	})
	if err != nil {
		return errors.Wrap(err, "prebuild")
	}

	logger.Donef("Successfully prebuild and pushed image %s", imageName)
	return nil
}
