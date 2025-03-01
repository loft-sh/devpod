package container

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}
	daemonCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "")
	daemonCmd.Flags().StringVar(&cmd.PlatformHost, "host", "", "")
	daemonCmd.Flags().StringVar(&cmd.NetworkHostname, "hostname", "", "")
	return daemonCmd
}

// Run runs the command logic
func (cmd *NetworkDaemonCmd) Run(ctx context.Context) error {
	// init kube config
	config := client.NewConfig()
	config.AccessKey = cmd.AccessKey
	config.Host = "https://" + cmd.PlatformHost
	config.Insecure = true
	baseClient := client.NewClientFromConfig(config)
	err := baseClient.RefreshSelf(context.TODO())
	if err != nil {
		return err
	}

	tsServer := ts.NewWorkspaceServer(&ts.WorkspaceServerConfig{
		AccessKey: cmd.AccessKey,
		Host:      ts.RemoveProtocol(cmd.PlatformHost),
		Hostname:  cmd.NetworkHostname,
		Client:    baseClient,
	}, log.Default) // FIXME: proper logging
	err = tsServer.Start(ctx)
	if err != nil {
		return fmt.Errorf("cannot start tsNet server: %w", err)
	}

	return nil
}
