//go:build darwin && !linux && !freebsd && !netbsd && !openbsd && !windows && !js
// +build darwin,!linux,!freebsd,!netbsd,!openbsd,!windows,!js

package beeep

import (
	"fmt"
	"os/exec"
)

// Alert displays a desktop notification and plays a default system sound.
func Alert(title, message, appIcon string) error {
	osa, err := exec.LookPath("osascript")
	if err != nil {
		return err
	}

	script := fmt.Sprintf("tell application \"System Events\" to display notification %q with title %q sound name \"default\"", message, title)
	cmd := exec.Command(osa, "-e", script)
	return cmd.Run()
}
