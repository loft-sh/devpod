package daemon

import (
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// NewCmd creates a new cobra command
func NewCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:    "daemon",
		Short:  "DevPod Pro Provider daemon commands",
		Args:   cobra.NoArgs,
		Hidden: true,
	}

	c.AddCommand(NewStartCmd(globalFlags))
	c.AddCommand(NewStatusCmd(globalFlags))
	c.AddCommand(NewNetcheckCmd(globalFlags))

	return c
}

func findProProvider(ctx context.Context, context, provider, host string, log log.Logger) (*config.Config, *providerpkg.ProviderConfig, error) {
	devPodConfig, err := config.LoadConfig(context, provider)
	if err != nil {
		return nil, nil, err
	}

	pCfg, err := workspace.ProviderFromHost(ctx, devPodConfig, host, log)
	if err != nil {
		return devPodConfig, nil, fmt.Errorf("load provider: %w", err)
	}

	return devPodConfig, pCfg, nil
}
