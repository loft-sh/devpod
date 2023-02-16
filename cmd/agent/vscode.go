package agent

import (
	"context"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/vscode"
	"github.com/spf13/cobra"
	"os"
)

type VSCodeCmd struct {
	User string

	Extensions []string
	Settings   string
}

// NewVSCodeCmd creates a new command
func NewVSCodeCmd() *cobra.Command {
	cmd := &VSCodeCmd{}
	vsCodeCmd := &cobra.Command{
		Use:   "vscode",
		Short: "Setups vscode inside the container",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	vsCodeCmd.Flags().StringSliceVar(&cmd.Extensions, "extension", []string{}, "The extensions to install")
	vsCodeCmd.Flags().StringVar(&cmd.Settings, "settings", "", "Json encoded settings to install")
	vsCodeCmd.Flags().StringVar(&cmd.User, "user", "", "The host to use")
	return vsCodeCmd
}

func (cmd *VSCodeCmd) Run(ctx context.Context) error {
	if cmd.Settings != "" {
		decompressed, err := compress.Decompress(cmd.Settings)
		if err != nil {
			return err
		}

		cmd.Settings = decompressed
	}

	vsCode := &vscode.VSCodeServer{}
	err := vsCode.Install(cmd.Extensions, cmd.Settings, cmd.User, os.Stdout)
	if err != nil {
		return err
	}

	return nil
}
