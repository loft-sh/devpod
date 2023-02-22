package agent

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// UpdateConfigCmd holds the cmd flags
type UpdateConfigCmd struct {
	*flags.GlobalFlags

	WorkspaceInfo string
}

// NewUpdateConfigCmd creates a new command
func NewUpdateConfigCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpdateConfigCmd{
		GlobalFlags: flags,
	}
	updateConfigCmd := &cobra.Command{
		Use:   "update-config",
		Short: "Updates the workspace config",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	updateConfigCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = updateConfigCmd.MarkFlagRequired("workspace-info")
	return updateConfigCmd
}

func (cmd *UpdateConfigCmd) Run(ctx context.Context) error {
	// get workspace
	_, err := writeWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return err
	}

	return nil
}
