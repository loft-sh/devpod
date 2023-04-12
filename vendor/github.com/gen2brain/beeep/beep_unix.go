//go:build linux || freebsd || netbsd || openbsd
// +build linux freebsd netbsd openbsd

package beeep

import (
	"errors"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// Constants
const (
	// This number represents the fixed frequency of the original PC XT's timer chip, which is approximately 1.193 MHz. This number
	// is divided with the desired frequency to obtain a counter value, that is subsequently fed into the timer chip, tied to the PC speaker.
	clockTickRate = 1193180

	// linux/kd.h, start sound generation (0 for off)
	kiocsound = 0x4B2F

	// linux/input-event-codes.h
	evSnd   = 0x12 // Event type
	sndTone = 0x02 // Sound
)

var (
	// DefaultFreq - frequency, in Hz, middle A
	DefaultFreq = 440.0
	// DefaultDuration - duration in milliseconds
	DefaultDuration = 200
)

// inputEvent represents linux/input.h event structure.
type inputEvent struct {
	Time  syscall.Timeval // time in seconds since epoch at which event occurred
	Type  uint16          // event type
	Code  uint16          // event code related to the event type
	Value int32           // event value related to the event type
}

// ioctl system call manipulates the underlying device parameters of special files.
func ioctl(fd, name, data uintptr) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, name, data)
	if e != 0 {
		return e
	}

	return nil
}

// Beep beeps the PC speaker (https://en.wikipedia.org/wiki/PC_speaker).
//
// On Linux it needs permission to access `/dev/tty0` or `/dev/input/by-path/platform-pcspkr-event-spkr` files for writing,
// and `pcspkr` module must be loaded. User must be in correct groups, usually `input` and/or `tty`.
//
// If it can not open device files, it will fallback to sending Bell character (https://en.wikipedia.org/wiki/Bell_character).
// For bell character in X11 terminals you can enable bell with `xset b on`. For console check `setterm` and `--blength` or `--bfreq` options.
//
// On macOS this just sends bell character. Enable `Audible bell` in Terminal --> Preferences --> Settings --> Advanced.
//
// On Windows it uses Beep function via syscall.
//
// On Web it plays hard coded beep sound.
func Beep(freq float64, duration int) error {
	if freq == 0 {
		freq = DefaultFreq
	} else if freq > 20000 {
		freq = 20000
	} else if freq < 0 {
		freq = DefaultFreq
	}

	if duration == 0 {
		duration = DefaultDuration
	}

	period := int(float64(clockTickRate) / freq)

	var evdev bool

	f, err := os.OpenFile("/dev/tty0", os.O_WRONLY, 0644)
	if err != nil {
		e := err
		f, err = os.OpenFile("/dev/input/by-path/platform-pcspkr-event-spkr", os.O_WRONLY, 0644)
		if err != nil {
			e = errors.New("beeep: " + e.Error() + "; " + err.Error())

			// Output the only beep we can
			_, err = os.Stdout.Write([]byte{7})
			if err != nil {
				return errors.New(e.Error() + "; " + err.Error())
			}

			return nil
		}

		evdev = true
	}

	defer f.Close()

	if evdev { // Use Linux evdev API
		ev := inputEvent{}
		ev.Type = evSnd
		ev.Code = sndTone
		ev.Value = int32(freq)

		d := *(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev))

		// Start beep
		f.Write(d[:])

		time.Sleep(time.Duration(duration) * time.Millisecond)

		ev.Value = 0
		d = *(*[unsafe.Sizeof(ev)]byte)(unsafe.Pointer(&ev))

		// Stop beep
		f.Write(d[:])
	} else { // Use ioctl
		// Start beep
		err = ioctl(f.Fd(), kiocsound, uintptr(period))
		if err != nil {
			return err
		}

		time.Sleep(time.Duration(duration) * time.Millisecond)

		// Stop beep
		err = ioctl(f.Fd(), kiocsound, uintptr(0))
		if err != nil {
			return err
		}
	}

	return nil
}
