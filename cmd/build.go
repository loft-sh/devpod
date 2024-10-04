package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/provider"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// BuildCmd holds the cmd flags
type BuildCmd struct {
	*flags.GlobalFlags
	provider.CLIOptions

	ProviderOptions []string

	SkipDelete bool
	Machine    string
}

// NewBuildCmd creates a new command
func NewBuildCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &BuildCmd{
		GlobalFlags: flags,
	}
	buildCmd := &cobra.Command{
		Use:   "build [flags] [workspace-path|workspace-name]",
		Short: "Builds a workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			// check permissions
			if !cmd.SkipPush && cmd.Repository != "" {
				err = image.CheckPushPermissions(cmd.Repository)
				if err != nil {
					return fmt.Errorf("cannot push to %s, please make sure you have push permissions to repository %s", cmd.Repository, cmd.Repository)
				}
			}

			// create a temporary workspace
			exists := workspace2.Exists(devPodConfig, args)
			sshConfigFile, err := os.CreateTemp("", "devpodssh.config")
			if err != nil {
				return err
			}
			sshConfigPath := sshConfigFile.Name()
			// defer removal of temporary ssh config file
			defer os.Remove(sshConfigPath)

			baseWorkspaceClient, err := workspace2.ResolveWorkspace(
				ctx,
				devPodConfig,
				"",
				nil,
				args,
				"",
				cmd.Machine,
				cmd.ProviderOptions,
				cmd.DevContainerImage,
				cmd.DevContainerPath,
				sshConfigPath,
				nil,
				cmd.UID,
				false,
				log.Default,
			)
			if err != nil {
				return err
			}

			// delete workspace if we have created it
			if exists == "" && !cmd.SkipDelete {
				defer func() {
					err = baseWorkspaceClient.Delete(ctx, client.DeleteOptions{Force: true})
					if err != nil {
						log.Default.Errorf("Error deleting workspace: %v", err)
					}
				}()
			}

			// check if regular workspace client
			workspaceClient, ok := baseWorkspaceClient.(client.WorkspaceClient)
			if !ok {
				return fmt.Errorf("building is currently not supported for proxy providers")
			}

			return cmd.Run(ctx, workspaceClient)
		},
	}

	buildCmd.Flags().StringVar(&cmd.DevContainerImage, "devcontainer-image", "", "The container image to use, this will override the devcontainer.json value in the project")
	buildCmd.Flags().StringVar(&cmd.DevContainerPath, "devcontainer-path", "", "The path to the devcontainer.json relative to the project")
	buildCmd.Flags().StringSliceVar(&cmd.ProviderOptions, "provider-option", []string{}, "Provider option in the form KEY=VALUE")
	buildCmd.Flags().BoolVar(&cmd.SkipDelete, "skip-delete", false, "If true will not delete the workspace after building it")
	buildCmd.Flags().StringVar(&cmd.Machine, "machine", "", "The machine to use for this workspace. The machine needs to exist beforehand or the command will fail. If the workspace already exists, this option has no effect")
	buildCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The repository to push to")
	buildCmd.Flags().StringSliceVar(&cmd.Platform, "platform", []string{}, "Set target platform for build")
	buildCmd.Flags().BoolVar(&cmd.SkipPush, "skip-push", false, "If true will not push the image to the repository, useful for testing")
	buildCmd.Flags().Var(&cmd.GitCloneStrategy, "git-clone-strategy", "The git clone strategy DevPod uses to checkout git based workspaces. Can be full (default), blobless, treeless or shallow")

	// TESTING
	buildCmd.Flags().BoolVar(&cmd.ForceBuild, "force-build", false, "TESTING ONLY")
	buildCmd.Flags().BoolVar(&cmd.ForceInternalBuildKit, "force-internal-buildkit", false, "TESTING ONLY")
	_ = buildCmd.Flags().MarkHidden("force-build")
	_ = buildCmd.Flags().MarkHidden("force-internal-buildkit")
	return buildCmd
}

func (cmd *BuildCmd) Run(ctx context.Context, client client.WorkspaceClient) error {
	// build workspace
	err := cmd.build(ctx, client, log.Default)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *BuildCmd) build(ctx context.Context, workspaceClient client.WorkspaceClient, log log.Logger) error {
	err := workspaceClient.Lock(ctx)
	if err != nil {
		return err
	}
	defer workspaceClient.Unlock()

	err = startWait(ctx, workspaceClient, true, log)
	if err != nil {
		return err
	}

	return cmd.buildAgentClient(ctx, workspaceClient, log)
}

func (cmd *BuildCmd) buildAgentClient(ctx context.Context, workspaceClient client.WorkspaceClient, log log.Logger) error {
	// compress info
	workspaceInfo, wInfo, err := workspaceClient.AgentInfo(cmd.CLIOptions)
	if err != nil {
		return err
	}

	// create container etc.
	log.Infof("Building devcontainer...")
	defer log.Debugf("Done building devcontainer")
	command := fmt.Sprintf("'%s' agent workspace build --workspace-info '%s'", workspaceClient.AgentPath(), workspaceInfo)
	if log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}

	// create pipes
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()
	defer stdinWriter.Close()

	// start machine on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		defer log.Debugf("Done executing up command")
		defer cancel()

		writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
		defer writer.Close()

		errChan <- agent.InjectAgentAndExecute(
			cancelCtx,
			func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
				return workspaceClient.Command(ctx, client.CommandOptions{
					Command: command,
					Stdin:   stdin,
					Stdout:  stdout,
					Stderr:  stderr,
				})
			},
			workspaceClient.AgentLocal(),
			workspaceClient.AgentPath(),
			workspaceClient.AgentURL(),
			true,
			command,
			stdinReader,
			stdoutWriter,
			writer,
			log.ErrorStreamOnly(),
			wInfo.InjectTimeout)
	}()

	// create container etc.
	_, err = tunnelserver.RunUpServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		workspaceClient.AgentInjectGitCredentials(),
		workspaceClient.AgentInjectDockerCredentials(),
		workspaceClient.WorkspaceConfig(),
		log,
	)
	if err != nil {
		return errors.Wrap(err, "run tunnel machine")
	}

	// wait until command finished
	return <-errChan
}
