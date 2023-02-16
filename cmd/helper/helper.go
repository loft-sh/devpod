package helper

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/helper/http"
	"github.com/loft-sh/devpod/cmd/helper/json"
	"github.com/loft-sh/devpod/cmd/helper/strings"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/spf13/cobra"
)

// NewHelperCmd returns a new command
func NewHelperCmd(flags *flags.GlobalFlags) *cobra.Command {
	helperCmd := &cobra.Command{
		Use:   "helper",
		Short: "DevPod Utility Commands",
		PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
			log.Default.MakeRaw()
			return nil
		},
		Hidden: true,
	}

	helperCmd.AddCommand(http.NewHTTPCmd(flags))
	helperCmd.AddCommand(json.NewJSONCmd(flags))
	helperCmd.AddCommand(strings.NewStringsCmd(flags))
	helperCmd.AddCommand(NewSSHServerCmd())
	helperCmd.AddCommand(NewSSHClientCmd())
	helperCmd.AddCommand(NewShellCmd())
	return helperCmd
}
