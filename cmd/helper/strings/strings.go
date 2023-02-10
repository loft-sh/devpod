package strings

import (
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/spf13/cobra"
)

// NewStringsCmd returns a new command
func NewStringsCmd(flags *flags.GlobalFlags) *cobra.Command {
	stringsCmd := &cobra.Command{
		Use:    "strings",
		Short:  "DevPod String Utility Commands",
		Hidden: true,
	}

	return stringsCmd
}
