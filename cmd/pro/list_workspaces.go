package pro

import (
	"bytes"
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// ListWorkspacesCmd holds the cmd flags
type ListWorkspacesCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host string
}

// NewListWorkspacesCmd creates a new command
func NewListWorkspacesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListWorkspacesCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "list-workspaces",
		Short:  "List Workspaces",
		Hidden: true,
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

func (cmd *ListWorkspacesCmd) Run(ctx context.Context, devPodConfig *config.Config, provider *provider.ProviderConfig) error {
	var buf bytes.Buffer
	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"listWorkspaces",
		provider.Exec.Proxy.List.Workspaces,
		devPodConfig.DefaultContext,
		nil,
		nil,
		devPodConfig.ProviderOptions(provider.Name),
		provider,
		nil,
		nil,
		&buf,
		nil,
		cmd.Log)
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	fmt.Println(buf.String())

	return nil
}
