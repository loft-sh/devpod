package util

import "time"

// WaitForChan races the channel closing against a timeout
func WaitForChan(channel <-chan error, timeout time.Duration) {
	select {
	case <-time.After(timeout):
		return
	case <-channel:
		return
	}
}
