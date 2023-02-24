//go:build (linux && !nodbus) || (freebsd && !nodbus) || (netbsd && !nodbus) || (openbsd && !nodbus)
// +build linux,!nodbus freebsd,!nodbus netbsd,!nodbus openbsd,!nodbus

package beeep

import (
	"errors"
	"os/exec"

	"github.com/godbus/dbus/v5"
)

// Notify sends desktop notification.
//
// On Linux it tries to send notification via D-Bus and it will fallback to `notify-send` binary.
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

	conn, err := dbus.SessionBus()
	if err != nil {
		return cmd()
	}
	obj := conn.Object("org.freedesktop.Notifications", dbus.ObjectPath("/org/freedesktop/Notifications"))

	call := obj.Call("org.freedesktop.Notifications.Notify", 0, "", uint32(0), appIcon, title, message, []string{}, map[string]dbus.Variant{}, int32(-1))
	if call.Err != nil {
		e := cmd()
		if e != nil {
			e := knotify()
			if e != nil {
				return errors.New("beeep: " + call.Err.Error() + "; " + e.Error())
			}
		}
	}

	return nil
}
