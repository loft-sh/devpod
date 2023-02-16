package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// ExecCmd holds the exec cmd flags
type ExecCmd struct {
	flags.GlobalFlags
}

// NewExecCmd creates a new command
func NewExecCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ExecCmd{
		GlobalFlags: *flags,
	}
	execCmd := &cobra.Command{
		Use:                "exec",
		DisableFlagParsing: true,
		Short:              "Executes a provider command",
		Long: `
Executes a provider command in a given workspace.

E.g.:
devpod provider exec aws status MY_WORKSPACE
devpod provider exec aws create MY_WORKSPACE
`,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return execCmd
}

// Run runs the command logic
func (cmd *ExecCmd) Run(ctx context.Context, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("expected exactly 3 arguments: PROVIDER COMMAND WORKSPACE")
	}

	devPodConfig, err := config.LoadConfig(cmd.Context)
	if err != nil {
		return err
	}

	devPodConfig.Contexts[devPodConfig.DefaultContext].DefaultProvider = args[0]
	workspaceConfig, provider, err := workspace.ResolveWorkspace(ctx, devPodConfig, nil, []string{args[2]}, "", log.Default)
	if err != nil {
		return err
	}

	// case server provider
	serverProvider, ok := provider.(provider2.ServerProvider)
	if ok {
		switch args[1] {
		case "create":
			err = serverProvider.Create(ctx, workspaceConfig, provider2.CreateOptions{})
			if err != nil {
				return err
			}
		case "delete":
			err = serverProvider.Delete(ctx, workspaceConfig, provider2.DeleteOptions{})
			if err != nil {
				return err
			}
		case "stop":
			err = serverProvider.Stop(ctx, workspaceConfig, provider2.StopOptions{})
			if err != nil {
				return err
			}
		case "start":
			err = serverProvider.Start(ctx, workspaceConfig, provider2.StartOptions{})
			if err != nil {
				return err
			}
		case "command":
			err = serverProvider.Command(ctx, workspaceConfig, provider2.CommandOptions{
				Command: strings.Join(args[3:], " "),
				Stdin:   os.Stdin,
				Stdout:  os.Stdout,
				Stderr:  os.Stderr,
			})
			if err != nil {
				return err
			}
		case "status":
			status, err := serverProvider.Status(ctx, workspaceConfig, provider2.StatusOptions{})
			if err != nil {
				return err
			}

			fmt.Println(status)
		}
	}

	return nil
}
