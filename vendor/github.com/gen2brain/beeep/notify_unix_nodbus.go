//go:build (linux && nodbus) || (freebsd && nodbus) || (netbsd && nodbus) || (openbsd && nodbus)
// +build linux,nodbus freebsd,nodbus netbsd,nodbus openbsd,nodbus

package beeep

import (
	"errors"
	"os/exec"
)

// Notify sends desktop notification.
func Notify(title, message, appIcon string) error {
	appIcon = pathAbs(appIcon)

	cmd := func() error {
		send, err := exec.LookPath("sw-notify-send")
		if err != nil {
			send, err = exec.LookPath("notify-send")
			if err != nil {
				return err
			}
		}

		c := exec.Command(send, title, message, "-i", appIcon)
		return c.Run()
	}

	knotify := func() error {
		send, err := exec.LookPath("kdialog")
		if err != nil {
			return err
		}
		c := exec.Command(send, "--title", title, "--passivepopup", message, "10", "--icon", appIcon)
		return c.Run()
	}

	err := cmd()
	if err != nil {
		e := knotify()
		if e != nil {
			return errors.New("beeep: " + err.Error() + "; " + e.Error())
		}
	}

	return nil
}
