package pro

import (
	"context"
	"fmt"
	"os"

	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/provider"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// DaemonStartCmd holds the devpod daemon flags
type DaemonStartCmd struct {
	*proflags.GlobalFlags

	Host string
	Log  log.Logger
}

// NewDaemonStartCmd creates a new command
func NewDaemonStartCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &DaemonStartCmd{
		GlobalFlags: flags,
		Log:         log.Default,
	}
	c := &cobra.Command{
		Use:   "daemon-start",
		Short: "Start the client daemon",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			devPodConfig, provider, err := findProProvider(cobraCmd.Context(), cmd.Context, cmd.Provider, cmd.Host, cmd.Log)
			if err != nil {
				return err
			}

			return cmd.Run(cobraCmd.Context(), devPodConfig, provider)
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *DaemonStartCmd) Run(ctx context.Context, devPodConfig *config.Config, provider *providerpkg.ProviderConfig) error {
	dir, err := ensureDaemonDir(devPodConfig.DefaultContext, provider.Name)
	if err != nil {
		return err
	}

	extraEnv := providerpkg.GetBaseEnvironment(devPodConfig.DefaultContext, provider.Name)
	extraEnv[platform.DaemonFolderEnv] = dir

	writer := devpodlog.PipeJSONStream(cmd.Log)
	defer writer.Close()

	err = clientimplementation.RunCommandWithBinaries(
		ctx,
		"startDaemon",
		provider.Exec.Proxy.Daemon.Start,
		devPodConfig.DefaultContext,
		nil,
		nil,
		devPodConfig.ProviderOptions(provider.Name),
		provider,
		extraEnv,
		nil,
		writer,
		writer,
		cmd.Log)
	if err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	return nil
}

func ensureDaemonDir(context, providerName string) (string, error) {
	tsDir, err := provider.GetDaemonDir(context, providerName)
	if err != nil {
		return "", fmt.Errorf("get daemon dir: %w", err)
	}

	err = os.MkdirAll(tsDir, 0o700)
	if err != nil {
		return tsDir, fmt.Errorf("make daemon dir: %w", err)
	}

	return tsDir, nil
}
