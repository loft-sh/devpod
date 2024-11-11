package helper

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/encoding"
	"github.com/spf13/cobra"
)

type GetWorkspaceUIDCommand struct {
	*flags.GlobalFlags
}

// NewGetWorkspaceUIDCmd creates a new command
func NewGetWorkspaceUIDCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetWorkspaceUIDCommand{
		GlobalFlags: flags,
	}
	shellCmd := &cobra.Command{
		Use:   "get-workspace-uid",
		Short: "Retrieves a workspace uid",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return shellCmd
}

func (cmd *GetWorkspaceUIDCommand) Run(ctx context.Context, args []string) error {
	fmt.Print(encoding.CreateNewUID("", ""))

	return nil
}
