package pro

import (
	"bytes"
	"context"
	"fmt"

	"github.com/loft-sh/devpod/cmd/agent"
	proflags "github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	devpodlog "github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/platform"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/ts"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// DaemonStatusCmd holds the devpod daemon flags
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

func (cmd *DaemonStatusCmd) Run(ctx context.Context, devPodConfig *config.Config, provider *provider.ProviderConfig) error {
	socket, err := ts.GetSocketForProvider(devPodConfig, provider.Name)
	if err != nil {
		return err
	}
	opts := devPodConfig.ProviderOptions(provider.Name)
	opts[platform.DaemonSocketEnv] = config.OptionValue{Value: socket}

	// ignore --debug because we tunnel json through stdio
	cmd.Log.SetLevel(logrus.InfoLevel)

	var buf, errBuf bytes.Buffer
	err = clientimplementation.RunCommandWithBinaries(
		ctx,
		"getDaemonStatus",
		provider.Exec.Proxy.Daemon.Status,
		devPodConfig.DefaultContext,
		nil,
		nil,
		opts,
		provider,
		nil,
		nil,
		&buf,
		&errBuf,
		cmd.Log)
	if err != nil {
		inner := ""
		lineObj, err2 := devpodlog.Unmarshal(errBuf.Bytes())
		if err2 == nil {
			inner = lineObj.Message
		}

		return fmt.Errorf("get daemon status: %w %s", err, inner)
	}
	fmt.Println(buf.String())

	return nil
}
