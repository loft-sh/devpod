package pro

import (
	"bytes"
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// ListProjectsCmd holds the cmd flags
type ListProjectsCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host string
}

// NewListProjectsCmd creates a new command
func NewListProjectsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListProjectsCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "list-projects",
		Short:  "List projects",
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *ListProjectsCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider, err := platform.ProviderFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}

	if !provider.IsProxyProvider() {
		return fmt.Errorf("only pro providers can list projects, provider \"%s\" is not a pro provider", provider.Name)
	}

	var buf bytes.Buffer
	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	if err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"listProjects",
		provider.Exec.Proxy.List.Projects,
		devPodConfig.DefaultContext,
		nil,
		nil,
		devPodConfig.ProviderOptions(provider.Name),
		provider,
		nil,
		nil,
		&buf,
		nil,
		cmd.Log); err != nil {
		return fmt.Errorf("watch workspaces with provider \"%s\": %w", provider.Name, err)
	}
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	fmt.Println(buf.String())

	return nil
}
