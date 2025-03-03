package helper

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/spf13/cobra"
)

type ShellCommand struct {
	Command string
	Login   bool
}

// NewShellCmd creates a new command
func NewShellCmd() *cobra.Command {
	cmd := &ShellCommand{}
	shellCmd := &cobra.Command{
		Use:   "sh",
		Short: "Executes a command in a shell",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	shellCmd.Flags().BoolVarP(&cmd.Login, "login", "l", false, "If login shell should be used")
	shellCmd.Flags().StringVarP(&cmd.Command, "command", "c", "", "Command to execute")
	return shellCmd
}

func (cmd *ShellCommand) Run(ctx context.Context, args []string) error {
	if cmd.Command == "" && len(args) == 0 {
		return nil
	} else if cmd.Command != "" && len(args) > 0 {
		return fmt.Errorf("either use -c or provide a script file")
	} else if len(args) > 1 {
		return fmt.Errorf("only a single script file can be used")
	}

	// load command from file
	if len(args) > 0 {
		content, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}

		cmd.Command = string(content)
	}

	return shell.RunEmulatedShell(ctx, cmd.Command, os.Stdin, os.Stdout, os.Stderr, os.Environ())
}
