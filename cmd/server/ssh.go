package server

import (
	"context"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// SSHCmd holds the configuration
type SSHCmd struct {
	*flags.GlobalFlags
}

// NewSSHCmd creates a new destroy command
func NewSSHCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: flags,
	}
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "SSH into the server",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background())
		},
	}

	return sshCmd
}

// Run runs the command logic
func (cmd *SSHCmd) Run(ctx context.Context, args []string) error {
	/*devPodConfig, err := config.LoadConfig(cmd.Context)
	if err != nil {
		return err
	}

	serverDir, err := provider.GetServersDir(devPodConfig.DefaultContext)
	if err != nil {
		return err
	}*/

	return nil
}
