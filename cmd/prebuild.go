package cmd

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"os"
)

// PrebuildCmd holds the cmd flags
type PrebuildCmd struct {
	*flags.GlobalFlags

	SkipDelete bool
	Repository string
	ForceBuild bool
}

// NewPrebuildCmd creates a new command
func NewPrebuildCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &PrebuildCmd{
		GlobalFlags: flags,
	}
	prebuildCmd := &cobra.Command{
		Use:   "prebuild",
		Short: "Prebuilds a workspace",
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			devPodConfig, err := config.LoadConfig(cmd.Context)
			if err != nil {
				return err
			}

			exists := workspace2.Exists(devPodConfig, args, log.Default)
			workspaceClient, err := workspace2.ResolveWorkspace(ctx, devPodConfig, nil, args, "", cmd.Provider, log.Default)
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

	prebuildCmd.Flags().BoolVar(&cmd.SkipDelete, "skip-delete", false, "If true will not delete the workspace after prebuilding it")
	prebuildCmd.Flags().BoolVar(&cmd.ForceBuild, "force-build", false, "If true will force build the image")
	prebuildCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The repository to push to")
	_ = prebuildCmd.MarkFlagRequired("repository")
	return prebuildCmd
}

func (cmd *PrebuildCmd) Run(ctx context.Context, client client.WorkspaceClient) error {
	// prebuild workspace
	err := cmd.prebuild(ctx, client, log.Default)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *PrebuildCmd) prebuild(ctx context.Context, workspaceClient client.WorkspaceClient, log log.Logger) error {
	err := startWait(ctx, workspaceClient, true, log)
	if err != nil {
		return err
	}

	agentClient, ok := workspaceClient.(client.AgentClient)
	if ok {
		return cmd.prebuildAgentClient(ctx, agentClient, log)
	}

	return fmt.Errorf("prebuilds are not supported for direct providers. Please use another provider instead")
}

func (cmd *PrebuildCmd) prebuildAgentClient(ctx context.Context, agentClient client.AgentClient, log log.Logger) error {
	// compress info
	workspaceInfo, err := agentClient.AgentInfo()
	if err != nil {
		return err
	}

	// create container etc.
	log.Infof("Prebuilding devcontainer...")
	defer log.Debugf("Done prebuilding devcontainer")
	command := fmt.Sprintf("%s agent workspace prebuild --workspace-info '%s'", agentClient.AgentPath(), workspaceInfo)
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

		writer := log.ErrorStreamOnly().Writer(logrus.DebugLevel, false)
		defer writer.Close()

		errChan <- agent.InjectAgentAndExecute(cancelCtx, func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return agentClient.Command(ctx, client.CommandOptions{
				Command: command,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			})
		}, agentClient.AgentPath(), agentClient.AgentURL(), true, command, stdinReader, stdoutWriter, writer, log.ErrorStreamOnly())
	}()

	// get workspace config
	workspaceConfig := agentClient.WorkspaceConfig()

	// create container etc.
	_, err = agent.RunTunnelServer(
		cancelCtx,
		stdoutReader,
		stdinWriter,
		false,
		string(workspaceConfig.Provider.Agent.InjectGitCredentials) == "true" && workspaceConfig.Source.GitRepository != "",
		string(workspaceConfig.Provider.Agent.InjectDockerCredentials) == "true",
		agentClient.WorkspaceConfig(),
		log,
	)
	if err != nil {
		return errors.Wrap(err, "run tunnel server")
	}

	// wait until command finished
	return <-errChan
}
