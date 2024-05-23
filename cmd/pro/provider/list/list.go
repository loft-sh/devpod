package list

import (
	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	c := &cobra.Command{
		Use:    "list",
		Short:  "DevPod Pro Provider List commands",
		Args:   cobra.NoArgs,
		Hidden: true,
	}

	c.AddCommand(NewProjectsCmd(globalFlags))
	c.AddCommand(NewTemplatesCmd(globalFlags))
	c.AddCommand(NewTemplateOptionsCmd(globalFlags))
	c.AddCommand(NewTemplateOptionsVersionCmd(globalFlags))
	return c
}
