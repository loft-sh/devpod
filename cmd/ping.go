package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/cmd/completion"
	"github.com/loft-sh/devpod/cmd/flags"
	client2 "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	workspace2 "github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type PingCmd struct {
	*flags.GlobalFlags
}

func NewPingCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &PingCmd{
		GlobalFlags: flags,
	}
	troubleshootCmd := &cobra.Command{
		Use:   "ping [workspace-path|workspace-name]",
		Short: "Pings the DevPod Pro workspace",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
		ValidArgsFunction: func(rootCmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completion.GetWorkspaceSuggestions(rootCmd, cmd.Context, cmd.Provider, args, toComplete, cmd.Owner, log.Default)
		},
		Hidden: true,
	}

	return troubleshootCmd
}

func (cmd *PingCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	client, err := workspace2.Get(ctx, devPodConfig, args, true, cmd.Owner, false, log.Default.ErrorStreamOnly())
	if err != nil {
		return err
	}

	daemonClient, ok := client.(client2.DaemonClient)
	if !ok {
		return fmt.Errorf("ping is only available for pro workspaces")
	}

	return daemonClient.Ping(ctx, os.Stdout)
}
