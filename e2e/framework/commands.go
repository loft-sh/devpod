package framework

import (
	"github.com/loft-sh/devpod/cmd"
)

func (f *Framework) ExecCommand(args []string) error {
	rootCmd := cmd.NewRootCmd()
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}
