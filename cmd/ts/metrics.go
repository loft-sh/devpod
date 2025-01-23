package ts

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/loft-sh/devpod/pkg/tailscale"
	"github.com/spf13/cobra"
)

type MetricsCmd struct {
	*TsNetFlags
}

type TsNetFlags struct {
	AccessKey       string
	PlatformHost    string
	NetworkHostname string
}

func (flags *TsNetFlags) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.AccessKey, "access-key", "", "")
	cmd.Flags().StringVar(&flags.PlatformHost, "host", "", "")
	cmd.Flags().StringVar(&flags.NetworkHostname, "hostname", "", "")
}

func NewMetricsCmd() *cobra.Command {
	cmd := &MetricsCmd{&TsNetFlags{}}
	var metricsCmd = &cobra.Command{
		Use:   "metrics print",
		Short: "Show Tailscale metrics",
		Long: strings.TrimSpace(`
			Prints current metric values in the Prometheus text exposition format
			
			For more information about Tailscale metrics, refer to
			https://tailscale.com/s/client-metrics
		`),
		RunE: cmd.Run,
	}
	cmd.TsNetFlags.ParseFlags(metricsCmd)
	return metricsCmd
}

func (cmd *MetricsCmd) Run(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	// Create network
	tsNet := tailscale.NewTSNet(&tailscale.TSNetConfig{
		AccessKey: cmd.AccessKey,
		Host:      tailscale.RemoveProtocol(cmd.PlatformHost),
		Hostname:  cmd.NetworkHostname,
	})
	// Run tailscale up and wait until we have a connected client
	done := make(chan bool)
	go func() {
		err := tsNet.Start(ctx, done)
		if err != nil {
			log.Fatalf("cannot start tsNet server: %v", err)
		}
	}()
	<-done

	// Get tailscale API client
	localClient, err := tsNet.LocalClient()
	if err != nil {
		return fmt.Errorf("cannot get local client: %w", err)
	}

	// Print metrics
	out, err := localClient.DaemonMetrics(ctx)
	if err != nil {
		return err
	}
	Stdout.Write(out)
	return nil

}
