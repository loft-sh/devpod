//go:build darwin && !linux && !freebsd && !netbsd && !openbsd && !windows && !js
// +build darwin,!linux,!freebsd,!netbsd,!openbsd,!windows,!js

package beeep

import (
	"fmt"
	"os/exec"
)

// Notify sends desktop notification.
//
// On macOS this executes AppleScript with `osascript` binary.
func Notify(title, message, appIcon string) error {
	osa, err := exec.LookPath("osascript")
	if err != nil {
		return err
	}

	script := fmt.Sprintf("display notification %q with title %q", message, title)
	cmd := exec.Command(osa, "-e", script)
	return cmd.Run()
}
