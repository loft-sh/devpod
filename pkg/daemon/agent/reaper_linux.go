//go:build linux
// +build linux

package agent

import reaper "github.com/ramr/go-reaper"

func RunProcessReaper() {
	reaper.Reap()
}
