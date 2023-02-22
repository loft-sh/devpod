package agent

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewAgentCmd returns a new root command
func NewAgentCmd(flags *flags.GlobalFlags) *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "DevPod Agent",
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			if flags.Silent {
				log.Default.SetLevel(logrus.FatalLevel)
			} else if flags.Debug {
				log.Default.SetLevel(logrus.DebugLevel)
			}

			log.Default.MakeRaw()
			return nil
		},
		Hidden: true,
	}

	agentCmd.AddCommand(NewUpCmd(flags))
	agentCmd.AddCommand(NewContainerTunnelCmd())
	agentCmd.AddCommand(NewDaemonCmd())
	agentCmd.AddCommand(NewDeleteCmd(flags))
	agentCmd.AddCommand(NewStopCmd(flags))
	agentCmd.AddCommand(NewStartCmd(flags))
	agentCmd.AddCommand(NewStatusCmd(flags))
	agentCmd.AddCommand(NewSetupContainerCmd())
	agentCmd.AddCommand(NewUpdateConfigCmd(flags))
	return agentCmd
}
