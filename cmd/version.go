package cmd

import (
	"fmt"

	"github.com/loft-sh/devpod/pkg/version"
	"github.com/spf13/cobra"
)

// VersionCmd holds the ws-tunnel cmd flags
type VersionCmd struct {
}

// NewVersionCmd creates a new ws-tunnel command
func NewVersionCmd() *cobra.Command {
	cmd := &VersionCmd{}
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the version",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	return versionCmd
}

// Run runs the command logic
func (cmd *VersionCmd) Run(_ *cobra.Command, _ []string) error {
	fmt.Print(version.GetVersion())
	return nil
}
