package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	platformdaemon "github.com/loft-sh/devpod/pkg/daemon/platform"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/sirupsen/logrus"

	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// StartCmd holds the devpod daemon flags
type StartCmd struct {
	*proflags.GlobalFlags

	Host string
	Log  log.Logger
}

// NewStartCmd creates a new command
func NewStartCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &StartCmd{
		GlobalFlags: flags,
	}
	c := &cobra.Command{
		Use:   "start",
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

func (cmd *StartCmd) Run(ctx context.Context, devPodConfig *config.Config, provider *providerpkg.ProviderConfig) error {
	dir, err := ensureDaemonDir(devPodConfig.DefaultContext, provider.Name)
	if err != nil {
		return err
	}

	loftConfigPath := filepath.Join(dir, "..", "loft-config.json")
	baseClient, err := client.InitClientFromPath(ctx, loftConfigPath)
	if err != nil {
		return err
	}
	userName := getUserName(baseClient.Self())
	if userName == "" {
		return fmt.Errorf("user name not set")
	}

	d, err := platformdaemon.Init(ctx, platformdaemon.InitConfig{
		RootDir:        dir,
		ProviderName:   provider.Name,
		Context:        devPodConfig.DefaultContext,
		UserName:       userName,
		PlatformClient: baseClient,
		Debug:          cmd.Debug,
	})
	if err != nil {
		return fmt.Errorf("init daemon: %w", err)
	}

	// NOTE: Do not remove, other processes rely on this for the startup sequence
	logInitialized()

	return d.Start()
}

func ensureDaemonDir(context, providerName string) (string, error) {
	tsDir, err := providerpkg.GetDaemonDir(context, providerName)
	if err != nil {
		return "", fmt.Errorf("get daemon dir: %w", err)
	}

	err = os.MkdirAll(tsDir, 0o700)
	if err != nil {
		return tsDir, fmt.Errorf("make daemon dir: %w", err)
	}

	return tsDir, nil
}

func logInitialized() {
	logger := log.NewStreamLogger(os.Stdout, os.Stderr, logrus.InfoLevel)
	logger.Done("Initialized daemon")
}

func getUserName(self *managementv1.Self) string {
	if self.Status.User != nil {
		return self.Status.User.Name
	}

	if self.Status.Team != nil {
		return self.Status.Team.Name
	}

	return self.Status.Subject
}
