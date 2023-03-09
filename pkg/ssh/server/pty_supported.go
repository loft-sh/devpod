//go:build !windows
// +build !windows

package server

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
)

func startPTY(cmd *exec.Cmd) (*os.File, error) {
	return pty.Start(cmd)
}

func setWinSize(f *os.File, w, h int) {
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}
