//go:build windows

package config

import "os/exec"

func PrepareCmdUser(cmd *exec.Cmd, userName string) error {
	return nil
}
