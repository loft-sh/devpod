package engine

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// LoginCmd holds the login cmd flags
type LoginCmd struct {
	flags.GlobalFlags

	Name string
}

// NewLoginCmd creates a new command
func NewLoginCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &LoginCmd{
		GlobalFlags: *flags,
	}
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Log into a DevPod Engine instance",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("please specify the engine url, e.g. devpod engine login my-engine.my-domain.com")
			}

			return cmd.Run(context.Background(), args[0])
		},
	}

	loginCmd.Flags().StringVar(&cmd.Name, "name", "", "Optional name how this DevPod engine will be referenced as")
	return loginCmd
}

// Run runs the command logic
func (cmd *LoginCmd) Run(ctx context.Context, url string) error {
	return nil
}
