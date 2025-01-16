// +build linux

package gonotify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Inotify is the low level wrapper around inotify_init(), inotify_add_watch() and inotify_rm_watch()
type Inotify struct {
	m        sync.Mutex
	fd       int
	f        *os.File
	watches  map[string]uint32
	rwatches map[uint32]string
}

// NewInotify creates new inotify instance
func NewInotify() (*Inotify, error) {
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)

	if err != nil {
		return nil, err
	}

	return &Inotify{
		fd:       fd,
		f:        os.NewFile(uintptr(fd), ""),
		watches:  make(map[string]uint32),
		rwatches: make(map[uint32]string),
	}, nil
}

// AddWatch adds given path to list of watched files / folders
func (i *Inotify) AddWatch(pathName string, mask uint32) error {
	w, err := syscall.InotifyAddWatch(i.fd, pathName, mask)

	if err != nil {
		return err
	}

	i.m.Lock()
	i.watches[pathName] = uint32(w)
	i.rwatches[uint32(w)] = pathName
	i.m.Unlock()
	return nil
}

// RmWd removes watch by watch descriptor
func (i *Inotify) RmWd(wd uint32) error {
	i.m.Lock()
	defer i.m.Unlock()

	pathName, ok := i.rwatches[wd]
	if !ok {
		return nil
	}

	_, err := syscall.InotifyRmWatch(i.fd, wd)
	if err != nil {
		return err
	}

	delete(i.watches, pathName)
	delete(i.rwatches, wd)
	return nil
}

// RmWatch removes watch by pathName
func (i *Inotify) RmWatch(pathName string) error {
	i.m.Lock()
	defer i.m.Unlock()

	wd, ok := i.watches[pathName]
	if !ok {
		return nil
	}

	_, err := syscall.InotifyRmWatch(i.fd, wd)
	if err != nil {
		return err
	}

	delete(i.watches, pathName)
	delete(i.rwatches, wd)
	return nil
}

// Read reads portion of InotifyEvents and may fail with an error
func (i *Inotify) Read() ([]InotifyEvent, error) {
	events := make([]InotifyEvent, 0, 1024)
	buf := make([]byte, 1024*(syscall.SizeofInotifyEvent+16))

	n, err := i.f.Read(buf)

	if err != nil {
		return events, err
	}

	if n < syscall.SizeofInotifyEvent {
		return events, fmt.Errorf("Short inotify read")
	}

	offset := 0

	for offset+syscall.SizeofInotifyEvent <= n {

		event := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
		namebuf := buf[offset+syscall.SizeofInotifyEvent : offset+syscall.SizeofInotifyEvent+int(event.Len)]

		offset += syscall.SizeofInotifyEvent + int(event.Len)

		name := strings.TrimRight(string(namebuf), "\x00")
		name = filepath.Join(i.rwatches[uint32(event.Wd)], name)
		events = append(events, InotifyEvent{
			Wd:     uint32(event.Wd),
			Name:   name,
			Mask:   event.Mask,
			Cookie: event.Cookie,
		})
	}

	return events, nil
}

// Close should be called when inotify is no longer needed in order to cleanup used resources.
func (i *Inotify) Close() error {
	i.m.Lock()
	defer i.m.Unlock()

	for _, w := range i.watches {
		_, err := syscall.InotifyRmWatch(i.fd, w)
		if err != nil {
			return err
		}
	}
	return i.f.Close()
}
