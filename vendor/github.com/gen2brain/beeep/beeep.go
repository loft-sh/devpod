// Package beeep provides a cross-platform library for sending desktop notifications and beeps.
package beeep

import (
	"errors"
	"path/filepath"
	"runtime"
)

var (
	// ErrUnsupported is returned when operating system is not supported.
	ErrUnsupported = errors.New("beeep: unsupported operating system: " + runtime.GOOS)
)

func pathAbs(path string) string {
	var err error
	var abs string

	if path != "" {
		abs, err = filepath.Abs(path)
		if err != nil {
			abs = path
		}
	}

	return abs
}
