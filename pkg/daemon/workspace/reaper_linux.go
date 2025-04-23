//go:build linux
// +build linux

package workspace

import reaper "github.com/ramr/go-reaper"

func RunProcessReaper() {
	reaper.Reap()
}
