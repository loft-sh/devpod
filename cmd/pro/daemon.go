package pro

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// DaemonCmd holds the devpod daemon flags
type DaemonCmd struct {
	*proflags.GlobalFlags

	Host string
	Log  log.Logger
}

// NewDaemonCmd creates a new command
func NewDaemonCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &DaemonCmd{
		GlobalFlags: flags,
		Log:         log.Default,
	}
	c := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the client daemon",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *DaemonCmd) Run(ctx context.Context, args []string) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider, err := platform.ProviderFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}

	if !provider.IsProxyProvider() {
		return fmt.Errorf("only pro providers can manage daemons, provider \"%s\" is not a pro provider", provider.Name)
	}

	tsDir, err := providerpkg.GetTailscaleDir(devPodConfig.DefaultContext, provider.Name)
	if err != nil {
		return err
	}

	// ensure tailscale dir
	err = os.Mkdir(tsDir, 0o700)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("ensure tailscale dir: %w", err)
	}

	// TODO: Should we move this into provider binary?
	daemon := ts.NewDaemon(tsDir, cmd.Log)

	return daemon.Start(ctx, cmd.Debug)
}
