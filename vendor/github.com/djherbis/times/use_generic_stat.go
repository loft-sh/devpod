// +build !windows,!linux

package times

import "os"

// Stat returns the Timespec for the given filename.
func Stat(name string) (Timespec, error) {
	return stat(name, os.Stat)
}

// Lstat returns the Timespec for the given filename, and does not follow Symlinks.
func Lstat(name string) (Timespec, error) {
	return stat(name, os.Lstat)
}

// StatFile returns the Timespec for the given *os.File.
func StatFile(file *os.File) (Timespec, error) {
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return getTimespec(fi), nil
}
