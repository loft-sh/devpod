package agent

import (
	"os"

	"github.com/loft-sh/devpod/cmd/agent/container"
	"github.com/loft-sh/devpod/cmd/agent/workspace"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var AgentExecutedAnnotation = "loft.sh/agent-executed"

// NewAgentCmd returns a new root command
func NewAgentCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "DevPod Agent",
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			return AgentPersistentPreRunE(cobraCmd, args, globalFlags)
		},
		Hidden: true,
	}

	agentCmd.AddCommand(workspace.NewWorkspaceCmd(globalFlags))
	agentCmd.AddCommand(container.NewContainerCmd(globalFlags))
	agentCmd.AddCommand(NewDaemonCmd(globalFlags))
	agentCmd.AddCommand(NewContainerTunnelCmd(globalFlags))
	agentCmd.AddCommand(NewGitCredentialsCmd(globalFlags))
	agentCmd.AddCommand(NewDockerCredentialsCmd(globalFlags))
	return agentCmd
}

func AgentPersistentPreRunE(cobraCmd *cobra.Command, args []string, globalFlags *flags.GlobalFlags) error {
	// get top level parent
	parent := cobraCmd
	for parent.Parent() != nil {
		parent = parent.Parent()
	}
	if parent.Annotations == nil {
		parent.Annotations = map[string]string{}
	}
	parent.Annotations[AgentExecutedAnnotation] = "true"

	if globalFlags.LogOutput == "json" {
		log.Default.SetFormat(log.JSONFormat)
	} else {
		log.Default.MakeRaw()
	}

	if globalFlags.Silent {
		log.Default.SetLevel(logrus.FatalLevel)
	} else if globalFlags.Debug {
		log.Default.SetLevel(logrus.DebugLevel)
	} else if os.Getenv(clientimplementation.DevPodDebug) == "true" {
		log.Default.SetLevel(logrus.DebugLevel)
	}

	if globalFlags.DevPodHome != "" {
		_ = os.Setenv(config.DEVPOD_HOME, globalFlags.DevPodHome)
	}

	return nil
}
