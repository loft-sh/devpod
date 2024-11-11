package helper

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/spf13/cobra"
)

type GetWorkspaceNameCommand struct {
	*flags.GlobalFlags
}

// NewGetWorkspaceNameCmd creates a new command
func NewGetWorkspaceNameCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetWorkspaceNameCommand{
		GlobalFlags: flags,
	}
	shellCmd := &cobra.Command{
		Use:   "get-workspace-name",
		Short: "Retrieves a workspace name",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	return shellCmd
}

func (cmd *GetWorkspaceNameCommand) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("workspace is missing")
	}

	_, name := file.IsLocalDir(args[0])
	workspaceID := workspace.ToID(name)
	fmt.Print(workspaceID)
	return nil
}
