package pro

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// WatchWorkspacesCmd holds the cmd flags
type WatchWorkspacesCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host          string
	Project       string
	FilterByOwner bool
}

// NewWatchWorkspacesCmd creates a new command
func NewWatchWorkspacesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &WatchWorkspacesCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "watch-workspaces",
		Short:  "Watch workspaces",
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")
	c.Flags().StringVar(&cmd.Project, "project", "", "The project to use")
	_ = c.MarkFlagRequired("project")
	c.Flags().BoolVar(&cmd.FilterByOwner, "filter-by-owner", true, "If true only shows workspaces of current owner")

	return c
}

func (cmd *WatchWorkspacesCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider, err := platform.ProviderFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}

	if !provider.IsProxyProvider() {
		return fmt.Errorf("only pro providers can watch workspaces, provider \"%s\" is not a pro provider", provider.Name)
	}

	opts := devPodConfig.ProviderOptions(provider.Name)
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if cmd.FilterByOwner {
		opts[providerpkg.LOFT_FILTER_BY_OWNER] = config.OptionValue{Value: "true"}
	}
	opts[providerpkg.LOFT_PROJECT] = config.OptionValue{Value: cmd.Project}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	go func() {
		<-sigChan
		cancel()
	}()

	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	err = clientimplementation.RunCommandWithBinaries(
		cancelCtx,
		"watchWorkspaces",
		provider.Exec.Proxy.Watch.Workspaces,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		provider,
		nil,
		nil,
		os.Stdout,
		log.Default.ErrorStreamOnly().Writer(logrus.ErrorLevel, false),
		cmd.Log)
	if err != nil {
		return fmt.Errorf("watch workspaces with provider \"%s\": %w", provider.Name, err)
	}

	return nil
}
