package cmd

import (
	"bytes"
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/devpod/pkg/log"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"os"
)

// BuildCmd holds the cmd flags
type BuildCmd struct {
	*flags.GlobalFlags

	SkipDelete bool
	Repository string
	Server     string
	ForceBuild bool
}

// NewBuildCmd creates a new command
func NewBuildCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &BuildCmd{
		GlobalFlags: flags,
	}
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Builds a workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			// check permissions
			err = image.CheckPushPermissions(cmd.Repository)
			if err != nil {
				return fmt.Errorf("cannot push to %s, please make sure you have push permissions to repository %s")
			}

			// create a temporary workspace
			exists := workspace2.Exists(devPodConfig, args, log.Default)
			workspaceClient, err := workspace2.ResolveWorkspace(ctx, devPodConfig, nil, args, "", cmd.Server, cmd.Provider, log.Default)
			if err != nil {
				return err
			}

			// delete workspace if we have created if
			if exists == "" {
				defer func() {
					err = workspaceClient.Delete(ctx, client.DeleteOptions{})
					if err != nil {
						log.Default.Errorf("Error deleting workspace: %v", err)
					}
				}()
			}

			return cmd.Run(ctx, workspaceClient)
		},
	}

	buildCmd.Flags().BoolVar(&cmd.SkipDelete, "skip-delete", false, "If true will not delete the workspace after building it")
	buildCmd.Flags().BoolVar(&cmd.ForceBuild, "force-build", false, "If true will force build the image")
	buildCmd.Flags().StringVar(&cmd.Server, "server", "", "The server to use for this workspace. The server needs to exist beforehand or the command will fail. If the workspace already exists, this option has no effect")
	buildCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The repository to push to")
	_ = buildCmd.MarkFlagRequired("repository")
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
	err := startWait(ctx, workspaceClient, true, false, log)
	if err != nil {
		return err
	}

	agentClient, ok := workspaceClient.(client.AgentClient)
	if ok {
		return cmd.buildAgentClient(ctx, agentClient, log)
	}

	return fmt.Errorf("builds are not supported for direct providers. Please use another provider instead")
}

func (cmd *BuildCmd) buildAgentClient(ctx context.Context, agentClient client.AgentClient, log log.Logger) error {
	// compress info
	workspaceInfo, err := agentClient.AgentInfo()
	if err != nil {
		return err
	}

	// create container etc.
	log.Infof("Building devcontainer...")
	defer log.Debugf("Done building devcontainer")
	command := fmt.Sprintf("%s agent workspace build --workspace-info '%s'", agentClient.AgentPath(), workspaceInfo)
	if log.GetLevel() == logrus.DebugLevel {
		command += " --debug"
	}
	if cmd.Repository != "" {
		command += fmt.Sprintf(" --repository '%s'", cmd.Repository)
	}
	if cmd.ForceBuild {
		command += " --force-build"
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

	// start server on stdio
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		defer log.Debugf("Done executing up command")
		defer cancel()

		buf := &bytes.Buffer{}
		err := agent.InjectAgentAndExecute(cancelCtx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return agentClient.Command(ctx, client.CommandOptions{
				Command: command,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			})
		}, agentClient.AgentPath(), agentClient.AgentURL(), true, command, stdinReader, stdoutWriter, buf, log.ErrorStreamOnly())
		if err != nil {
			errChan <- errors.Wrapf(err, "%s", buf.String())
		} else {
			errChan <- nil
		}
	}()

	// get workspace config
	workspaceConfig := agentClient.WorkspaceConfig()
	agentConfig := agentClient.AgentConfig()

	// create container etc.
	_, err = agent.RunTunnelServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		false,
		string(agentConfig.InjectGitCredentials) == "true" && workspaceConfig.Source.GitRepository != "",
		string(agentConfig.InjectDockerCredentials) == "true",
		agentClient.WorkspaceConfig(),
		log,
	)
	if err != nil {
		return errors.Wrap(err, "run tunnel server")
	}

	// wait until command finished
	return <-errChan
}
