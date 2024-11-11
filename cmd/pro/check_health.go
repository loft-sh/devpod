package pro

import (
	"bytes"
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/agent"
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// CheckHealthCmd holds the cmd flags
type CheckHealthCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Host string
}

// NewCheckHealthCmd creates a new command
func NewCheckHealthCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CheckHealthCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:    "check-health",
		Short:  "Check platform health",
		Hidden: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			root := cmd.Root()
			if root == nil {
				return
			}
			if root.Annotations == nil {
				root.Annotations = map[string]string{}
			}
			// Don't print debug message
			root.Annotations[agent.AgentExecutedAnnotation] = "true"
		},
	}

	c.Flags().StringVar(&cmd.Host, "host", "", "The pro instance to use")
	_ = c.MarkFlagRequired("host")

	return c
}

func (cmd *CheckHealthCmd) Run(ctx context.Context) error {
	devPodConfig, err := config.LoadConfig(cmd.Context, cmd.Provider)
	if err != nil {
		return err
	}

	provider, err := platform.ProviderFromHost(ctx, devPodConfig, cmd.Host, cmd.Log)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}

	if !provider.IsProxyProvider() {
		return fmt.Errorf("only pro providers can check health, provider \"%s\" is not a pro provider", provider.Name)
	}

	var buf bytes.Buffer
	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	if err := clientimplementation.RunCommandWithBinaries(
		ctx,
		"health",
		provider.Exec.Proxy.Health,
		devPodConfig.DefaultContext,
		nil,
		nil,
		devPodConfig.ProviderOptions(provider.Name),
		provider,
		nil,
		nil,
		&buf,
		cmd.Log.Writer(logrus.ErrorLevel, true),
		cmd.Log); err != nil {
		return fmt.Errorf("check health with provider \"%s\": %w", provider.Name, err)
	}
	if err != nil {
		return fmt.Errorf("check health: %w", err)
	}

	fmt.Println(buf.String())

	return nil
}
