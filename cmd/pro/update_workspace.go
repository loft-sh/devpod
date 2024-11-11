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

// UpdateWorkspaceCmd holds the cmd flags
type UpdateWorkspaceCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host     string
	Instance string
}

// NewListworkspacesCmd creates a new command
func NewUpdateWorkspaceCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpdateWorkspaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "update-workspace",
		Short:  "Update workspace instance",
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")
	c.Flags().StringVar(&cmd.Instance, "instance", "", "The workspace instance to update")
	_ = c.MarkFlagRequired("instance")

	return c
}

func (cmd *UpdateWorkspaceCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider, err := platform.ProviderFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}

	if !provider.IsProxyProvider() {
		return fmt.Errorf("only pro providers can update workspaces, provider \"%s\" is not a pro provider", provider.Name)
	}

	opts := devPodConfig.ProviderOptions(provider.Name)
	opts[platform.WorkspaceInstanceEnv] = config.OptionValue{Value: cmd.Instance}

	var buf bytes.Buffer
	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	if err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"updateWorkspace",
		provider.Exec.Proxy.Update.Workspace,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		provider,
		nil,
		nil,
		&buf,
		cmd.Log.ErrorStreamOnly().Writer(logrus.ErrorLevel, true),
		cmd.Log); err != nil {
		return fmt.Errorf("update workspace with provider \"%s\": %w", provider.Name, err)
	}
	if err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}

	fmt.Println(buf.String())

	return nil
}
