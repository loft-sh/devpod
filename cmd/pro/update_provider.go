package pro

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	providercmd "github.com/loft-sh/devpod/cmd/provider"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/workspace"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// UpdateProviderCmd holds the cmd flags
type UpdateProviderCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host     string
	Instance string
}

// NewUpdateProviderCmd creates a new command
func NewUpdateProviderCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpdateProviderCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "update-provider [new-version]",
		Short:  "Update platform provider",
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *UpdateProviderCmd) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("new version is missing")
	}
	newVersion := args[0]

	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider, err := platform.ProviderFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}
	providerSource, err := workspace.ResolveProviderSource(devPodConfig, provider.Name, cmd.Log)
	if err != nil {
		return fmt.Errorf("resolve provider source %s: %w", provider.Name, err)
	}
	splitted := strings.Split(providerSource, "@")
	if len(splitted) == 0 {
		return fmt.Errorf("no provider source found %s", providerSource)
	}
	providerSource = splitted[0] + "@" + newVersion

	_, err = workspace.UpdateProvider(devPodConfig, provider.Name, providerSource, cmd.Log)
	if err != nil {
		return fmt.Errorf("update provider %s: %w", provider.Name, err)
	}

	err = providercmd.ConfigureProvider(ctx, provider, devPodConfig.DefaultContext, []string{}, true, true, true, nil, log.Discard)
	if err != nil {
		return fmt.Errorf("configure provider, please retry with 'devpod provider use %s --reconfigure': %w", provider.Name, err)
	}

	return nil
}
