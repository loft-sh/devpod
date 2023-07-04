package workspace

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// BuildCmd holds the cmd flags
type BuildCmd struct {
	*flags.GlobalFlags

	Repository    string
	WorkspaceInfo string
	Platforms     []string
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
	buildCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The repository to push to")
	buildCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	buildCmd.Flags().StringSliceVar(&cmd.Platforms, "platform", []string{}, "Set target platform for build")
	_ = buildCmd.MarkFlagRequired("workspace-info")
	_ = buildCmd.MarkFlagRequired("repository")
	return buildCmd
}

// Run runs the command logic
func (cmd *BuildCmd) Run(ctx context.Context) error {
	if cmd.Repository == "" {
		return fmt.Errorf("repository needs to be specified")
	}

	// write workspace info
	shouldExit, workspaceInfo, err := agent.WriteWorkspaceInfoAndDeleteOld(cmd.WorkspaceInfo, func(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
		return deleteWorkspace(ctx, workspaceInfo, log)
	}, log.Default.ErrorStreamOnly())
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

	runner, err := CreateRunner(workspaceInfo, logger)
	if err != nil {
		return err
	}

	// if there is no platform specified, we use empty to let
	// the builder find out itself.
	if len(cmd.Platforms) == 0 {
		cmd.Platforms = []string{""}
	}

	// build and push images
	for _, platform := range cmd.Platforms {
		// build the image
		imageName, err := runner.Build(ctx, config.BuildOptions{
			PushRepository: cmd.Repository,
			Platform:       platform,
		})
		if err != nil {
			logger.Errorf("Error building image: %v", err)
			return errors.Wrap(err, "build")
		}

		logger.Donef("Successfully build and pushed image %s", imageName)
	}

	return nil
}

func deleteWorkspace(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	err := removeContainer(ctx, workspaceInfo, log)
	if err != nil {
		return errors.Wrap(err, "remove container")
	}

	_ = os.RemoveAll(workspaceInfo.Origin)
	return nil
}
