package pro

import (
	"bytes"
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// VersionCmd holds the cmd flags
type VersionCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host string
}

// NewVersionCmd creates a new command
func NewVersionCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VersionCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "version",
		Short:  "Get version",
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *VersionCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider, err := platform.ProviderFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}

	if !provider.IsProxyProvider() {
		return fmt.Errorf("only pro providers can get version, provider \"%s\" is not a pro provider", provider.Name)
	}

	opts := devPodConfig.ProviderOptions(provider.Name)
	opts[providerpkg.PROVIDER_ID] = config.OptionValue{Value: provider.Name}
	opts[providerpkg.PROVIDER_CONTEXT] = config.OptionValue{Value: cmd.Context}

	var buf bytes.Buffer
	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	err = clientimplementation.RunCommandWithBinaries(
		ctx,
		"getVersion",
		provider.Exec.Proxy.Get.Version,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		provider,
		nil,
		nil,
		&buf,
		nil,
		cmd.Log)
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}

	fmt.Print(buf.String())

	return nil
}
