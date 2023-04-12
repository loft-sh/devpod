//go:build linux || freebsd || netbsd || openbsd
// +build linux freebsd netbsd openbsd

package beeep

// Alert displays a desktop notification and plays a beep.
func Alert(title, message, appIcon string) error {
	if err := Notify(title, message, appIcon); err != nil {
		return err
	}
	return Beep(DefaultFreq, DefaultDuration)
}
