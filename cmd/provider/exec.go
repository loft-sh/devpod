package provider

import (
	"context"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
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

	var (
		client client2.WorkspaceClient
	)
	if args[1] == "create" {
		client, err = workspace.ResolveWorkspace(ctx, devPodConfig, nil, []string{args[2]}, "", args[0], log.Default)
	} else {
		client, err = workspace.GetWorkspace(ctx, devPodConfig, nil, []string{args[2]}, log.Default)
	}
	if err != nil {
		return err
	}

	// case server provider
	switch args[1] {
	case "create":
		err = client.Create(ctx, client2.CreateOptions{})
		if err != nil {
			return err
		}
	case "delete":
		err = client.Delete(ctx, client2.DeleteOptions{})
		if err != nil {
			return err
		}
	case "stop":
		err = client.Stop(ctx, client2.StopOptions{})
		if err != nil {
			return err
		}
	case "start":
		err = client.Start(ctx, client2.StartOptions{})
		if err != nil {
			return err
		}
	case "command":
		err = client.Command(ctx, client2.CommandOptions{
			Command: strings.Join(args[3:], " "),
			Stdin:   os.Stdin,
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
		if err != nil {
			return err
		}
	case "status":
		status, err := client.Status(ctx, client2.StatusOptions{})
		if err != nil {
			return err
		}

		fmt.Println(status)
	}

	return nil
}
