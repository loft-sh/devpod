package agent

import (
	"context"
	"github.com/spf13/cobra"
)

// WatchCmd holds the up cmd flags
type WatchCmd struct{}

// NewWatchCmd creates a new ssh command
func NewWatchCmd() *cobra.Command {
	cmd := &WatchCmd{}
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Watches for activity and stops the server due to inactivity",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	return watchCmd
}

func (cmd *WatchCmd) Run(ctx context.Context) error {

	return nil
}
