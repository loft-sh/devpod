package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GetWorkspaceConfigCommand struct {
	*flags.GlobalFlags

	timeout  time.Duration
	maxDepth int
}
type GetWorkspaceConfigCommandResult struct {
	IsImage         bool     `json:"isImage"`
	IsGitRepository bool     `json:"isGitRepository"`
	IsLocal         bool     `json:"isLocal"`
	ConfigPaths     []string `json:"configPaths"`
}

// NewGetWorkspaceConfigCommand creates a new command
func NewGetWorkspaceConfigCommand(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetWorkspaceConfigCommand{
		GlobalFlags: flags,
	}
	shellCmd := &cobra.Command{
		Use:   "get-workspace-config",
		Short: "Retrieves a workspace config",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
			if err != nil {
				return err
			}

			if cmd.maxDepth < 0 {
				log.Default.Debugf("--max-depth was %d, setting to 0", cmd.maxDepth)
				cmd.maxDepth = 0
			}

			return cmd.Run(cobraCmd.Context(), devPodConfig, args)
		},
	}

	shellCmd.Flags().DurationVar(&cmd.timeout, "timeout", 10*time.Second, "Timeout for the command, 10 seconds by default")
	shellCmd.Flags().IntVar(&cmd.maxDepth, "max-depth", 3, "Maximum depth to search for devcontainer files")

	return shellCmd
}

func (cmd *GetWorkspaceConfigCommand) Run(ctx context.Context, devPodConfig *config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("workspace source is missing")
	}
	rawSource := args[0]

	level := log.Default.GetLevel()
	if cmd.GlobalFlags.Debug {
		level = logrus.DebugLevel
	}
	var logger log.Logger = log.NewStdoutLogger(os.Stdin, os.Stdout, os.Stderr, level)
	if os.Getenv("DEVPOD_UI") == "true" {
		logger = log.Discard
	}
	logger.Debugf("Resolving devcontainer config for source: %s", rawSource)

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	done := make(chan *devcontainer.GetWorkspaceConfigResult, 1)
	errChan := make(chan error, 1)

	tmpDir, err := os.MkdirTemp("", "devpod")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	go func() {
		result, err := devcontainer.FindDevcontainerFiles(ctx, rawSource, tmpDir, cmd.maxDepth, devPodConfig.ContextOption(config.ContextOptionSSHStrictHostKeyChecking) == "true", logger)
		if err != nil {
			errChan <- err
			return
		}
		done <- result
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("unable to find devcontainer files: %w", err)
	case <-ctx.Done():
		return fmt.Errorf("timeout while searching for devcontainer files")
	case result := <-done:
		out, err := json.Marshal(result)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}
	defer close(done)

	return nil
}
