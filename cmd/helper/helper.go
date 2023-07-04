package helper

import (
	"os"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/helper/http"
	"github.com/loft-sh/devpod/cmd/helper/json"
	"github.com/loft-sh/devpod/cmd/helper/strings"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewHelperCmd returns a new command
func NewHelperCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	helperCmd := &cobra.Command{
		Use:   "helper",
		Short: "DevPod Utility Commands",
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
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
		},
		Hidden: true,
	}

	helperCmd.AddCommand(http.NewHTTPCmd(globalFlags))
	helperCmd.AddCommand(json.NewJSONCmd(globalFlags))
	helperCmd.AddCommand(strings.NewStringsCmd(globalFlags))
	helperCmd.AddCommand(NewSSHServerCmd(globalFlags))
	helperCmd.AddCommand(NewGetWorkspaceNameCmd(globalFlags))
	helperCmd.AddCommand(NewGetWorkspaceConfigCommand(globalFlags))
	helperCmd.AddCommand(NewGetProviderNameCmd(globalFlags))
	helperCmd.AddCommand(NewCheckProviderUpdateCmd(globalFlags))
	helperCmd.AddCommand(NewSSHClientCmd())
	helperCmd.AddCommand(NewShellCmd())
	return helperCmd
}
