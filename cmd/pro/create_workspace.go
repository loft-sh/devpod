package pro

import (
	"bytes"
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// CreateWorkspaceCmd holds the cmd flags
type CreateWorkspaceCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host     string
	Instance string
}

// NewCreateWorkspaceCmd creates a new command
func NewCreateWorkspaceCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateWorkspaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "create-workspace",
		Short:  "Create workspace instance",
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
	c.Flags().StringVar(&cmd.Instance, "instance", "", "The workspace instance to create")
	_ = c.MarkFlagRequired("instance")

	return c
}

func (cmd *CreateWorkspaceCmd) Run(ctx context.Context, devPodConfig *config.Config, provider *provider.ProviderConfig) error {
	opts := devPodConfig.ProviderOptions(provider.Name)
	opts[platform.WorkspaceInstanceEnv] = config.OptionValue{Value: cmd.Instance}

	var buf bytes.Buffer
	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"createWorkspace",
		provider.Exec.Proxy.Create.Workspace,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		provider,
		nil,
		nil,
		&buf,
		cmd.Log.ErrorStreamOnly().Writer(logrus.ErrorLevel, true),
		cmd.Log)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	fmt.Println(buf.String())

	return nil
}
