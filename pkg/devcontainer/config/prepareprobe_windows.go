//go:build windows

package config

import "os/exec"

func PrepareProbe(cmd *exec.Cmd, userName string) error {
	return nil
}
