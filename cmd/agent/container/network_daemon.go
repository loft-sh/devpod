package container

import (
	"context"
	"fmt"
	"net"

	"github.com/loft-sh/devpod/cmd/flags"
	sshServer "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/tailscale"
	"github.com/spf13/cobra"
)

// NetworkDaemonCmd holds the cmd flags
type NetworkDaemonCmd struct {
	*flags.GlobalFlags

	AccessKey       string
	PlatformHost    string
	NetworkHostname string
}

// NewDaemonCmd creates a new command
func NewNetworkDaemonCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &NetworkDaemonCmd{
		GlobalFlags: flags,
	}
	daemonCmd := &cobra.Command{
		Use:   "network-daemon",
		Short: "Starts tailscale network daemon",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	daemonCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "")
	daemonCmd.Flags().StringVar(&cmd.PlatformHost, "host", "", "")
	daemonCmd.Flags().StringVar(&cmd.NetworkHostname, "hostname", "", "")
	return daemonCmd
}

// Run runs the command logic
func (cmd *NetworkDaemonCmd) Run(_ *cobra.Command, _ []string) error {
	tsNet := tailscale.NewTSNet(&tailscale.TSNetConfig{
		AccessKey: cmd.AccessKey,
		Host:      tailscale.RemoveProtocol(cmd.PlatformHost),
		Hostname:  cmd.NetworkHostname,
		PortHandlers: map[string]func(net.Listener){
			fmt.Sprintf("%d", sshServer.DefaultPort):     tailscale.ReverseProxyHandler(fmt.Sprintf("127.0.0.1:%d", sshServer.DefaultPort)),
			fmt.Sprintf("%d", sshServer.DefaultUserPort): tailscale.ReverseProxyHandler(fmt.Sprintf("127.0.0.1:%d", sshServer.DefaultUserPort)),
		},
	})
	if err := tsNet.Start(context.TODO(), nil); err != nil {
		return fmt.Errorf("cannot start tsNet server: %w", err)
	}

	return nil
}
