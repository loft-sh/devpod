package helper

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/cmd/helper/http"
	"github.com/spf13/cobra"
)

// NewHelperCmd returns a new command
func NewHelperCmd(flags *flags.GlobalFlags) *cobra.Command {
	helperCmd := &cobra.Command{
		Use:    "helper",
		Short:  "DevPod Utility Commands",
		Hidden: true,
	}

	helperCmd.AddCommand(http.NewHTTPCmd(flags))
	helperCmd.AddCommand(NewSSHServerCmd())
	helperCmd.AddCommand(NewSSHClientCmd())
	return helperCmd
}
