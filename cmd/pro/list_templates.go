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

// ListTemplatesCmd holds the cmd flags
type ListTemplatesCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host    string
	Project string
}

// NewListTemplatesCmd creates a new command
func NewListTemplatesCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListTemplatesCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "list-templates",
		Short:  "List templates",
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")
	c.Flags().StringVar(&cmd.Project, "project", "", "The project to use")
	_ = c.MarkFlagRequired("project")

	return c
}

func (cmd *ListTemplatesCmd) Run(ctx context.Context) error {
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

	opts := devPodConfig.ProviderOptions(provider.Name)
	opts[platform.ProjectEnv] = config.OptionValue{Value: cmd.Project}

	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)
	var buf bytes.Buffer
	if err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"listTemplates",
		provider.Exec.Proxy.List.Templates,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		provider,
		nil,
		nil,
		&buf,
		nil,
		cmd.Log); err != nil {
		return fmt.Errorf("list templates with provider \"%s\": %w", provider.Name, err)
	}
	if err != nil {
		return fmt.Errorf("list templates: %w", err)
	}

	fmt.Println(buf.String())

	return nil
}
