package pro

import (
	"context"
	"encoding/json"
	"fmt"
	platformdaemon "github.com/loft-sh/devpod/pkg/platform/daemon"

	"github.com/loft-sh/devpod/cmd/agent"
	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/config"
	providerpkg "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// DaemonStatusCmd holds the DevPod daemon flags
type DaemonStatusCmd struct {
	*proflags.GlobalFlags

	Host string
	Log  log.Logger
}

// NewDaemonStatusCmd creates a new command
func NewDaemonStatusCmd(flags *proflags.GlobalFlags) *cobra.Command {
	cmd := &DaemonStatusCmd{
		GlobalFlags: flags,
		Log:         log.Default,
	}
	c := &cobra.Command{
		Use:   "daemon-status",
		Short: "Get the status of the daemon",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			devPodConfig, provider, err := findProProvider(cobraCmd.Context(), cmd.Context, cmd.Provider, cmd.Host, cmd.Log)
			if err != nil {
				return err
			}

			return cmd.Run(cobraCmd.Context(), devPodConfig, provider)
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

func (cmd *DaemonStatusCmd) Run(ctx context.Context, devPodConfig *config.Config, provider *providerpkg.ProviderConfig) error {
	dir, err := providerpkg.GetDaemonDir(devPodConfig.DefaultContext, provider.Name)
	if err != nil {
		return fmt.Errorf("get daemon dir: %w", err)
	}

	client := platformdaemon.NewClient(dir, provider.Name)
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}
	out, err := json.Marshal(status)
	if err != nil {
		return err
	}

	fmt.Print(string(out))

	return nil
}
