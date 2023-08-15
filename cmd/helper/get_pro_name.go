package helper

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
)

type GetProNameCommand struct {
	*flags.GlobalFlags
}

// NewGetProNameCmd creates a new command
func NewGetProNameCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetProNameCommand{
		GlobalFlags: flags,
	}
	shellCmd := &cobra.Command{
		Use:   "get-pro-name",
		Short: "Retrieves a pro instance name",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return shellCmd
}

func (cmd *GetProNameCommand) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("pro instance url is missing")
	}

	fmt.Print(workspace.ToProInstanceID(args[0]))
	return nil
}
