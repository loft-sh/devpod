package agent

import (
	"context"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/vscode"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

type OpenVSCodeCmd struct {
	User string
	Host string
	Port string

	Extensions []string
	Settings   string
}

// NewOpenVSCodeCmd creates a new command
func NewOpenVSCodeCmd() *cobra.Command {
	cmd := &OpenVSCodeCmd{}
	openVSCodeCmd := &cobra.Command{
		Use:   "openvscode",
		Short: "Starts openvscode inside the container",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	openVSCodeCmd.Flags().StringSliceVar(&cmd.Extensions, "extension", []string{}, "The extensions to install")
	openVSCodeCmd.Flags().StringVar(&cmd.Settings, "settings", "", "Json encoded settings to install")
	openVSCodeCmd.Flags().StringVar(&cmd.User, "user", "", "The host to use")
	openVSCodeCmd.Flags().StringVar(&cmd.Host, "host", "0.0.0.0", "The host to use")
	openVSCodeCmd.Flags().StringVar(&cmd.Port, "port", strconv.Itoa(vscode.DefaultVSCodePort), "The port to listen to")
	return openVSCodeCmd
}

func (cmd *OpenVSCodeCmd) Run(ctx context.Context) error {
	if cmd.Settings != "" {
		decompressed, err := compress.Decompress(cmd.Settings)
		if err != nil {
			return err
		}

		cmd.Settings = decompressed
	}

	openVSCode := &vscode.OpenVSCodeServer{}
	err := openVSCode.InstallAndStart(cmd.Extensions, cmd.Settings, cmd.User, cmd.Host, cmd.Port, os.Stdout)
	if err != nil {
		return err
	}

	return nil
}
