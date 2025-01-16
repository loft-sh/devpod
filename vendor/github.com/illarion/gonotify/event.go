// +build linux

package gonotify

import (
	"fmt"
	"strings"
	"syscall"
)

const (
	IN_ACCESS        = uint32(syscall.IN_ACCESS)        // File was accessed
	IN_ATTRIB        = uint32(syscall.IN_ATTRIB)        // Metadata changed
	IN_CLOSE_WRITE   = uint32(syscall.IN_CLOSE_WRITE)   // File opened for writing was closed.
	IN_CLOSE_NOWRITE = uint32(syscall.IN_CLOSE_NOWRITE) // File or directory not opened for writing was closed.
	IN_CREATE        = uint32(syscall.IN_CREATE)        // File/directory created in watched directory
	IN_DELETE        = uint32(syscall.IN_DELETE)        // File/directory deleted from watched directory.
	IN_DELETE_SELF   = uint32(syscall.IN_DELETE_SELF)   // Watched file/directory was itself deleted.
	IN_MODIFY        = uint32(syscall.IN_MODIFY)        // File was modified
	IN_MOVE_SELF     = uint32(syscall.IN_MOVE_SELF)     // Watched file/directory was itself moved.
	IN_MOVED_FROM    = uint32(syscall.IN_MOVED_FROM)    // Generated for the directory containing the old filename when a file is renamed.
	IN_MOVED_TO      = uint32(syscall.IN_MOVED_TO)      // Generated for the directory containing the new filename when a file is renamed.
	IN_OPEN          = uint32(syscall.IN_OPEN)          // File or directory was opened.

	IN_ALL_EVENTS = uint32(syscall.IN_ALL_EVENTS) // bit mask of all of the above events.
	IN_MOVE       = uint32(syscall.IN_MOVE)       // Equates to IN_MOVED_FROM | IN_MOVED_TO.
	IN_CLOSE      = uint32(syscall.IN_CLOSE)      // Equates to IN_CLOSE_WRITE | IN_CLOSE_NOWRITE.

	/* The following further bits can be specified in mask when calling Inotify.AddWatch() */

	IN_DONT_FOLLOW = uint32(syscall.IN_DONT_FOLLOW) // Don't dereference pathname if it is a symbolic link.
	IN_EXCL_UNLINK = uint32(syscall.IN_EXCL_UNLINK) // Don't generate events for children if they have been unlinked from the directory.
	IN_MASK_ADD    = uint32(syscall.IN_MASK_ADD)    // Add (OR) the events in mask to the watch mask
	IN_ONESHOT     = uint32(syscall.IN_ONESHOT)     // Monitor the filesystem object corresponding to pathname for one event, then remove from watch list.
	IN_ONLYDIR     = uint32(syscall.IN_ONLYDIR)     // Watch pathname only if it is a directory.

	/* The following bits may be set in the mask field returned by Inotify.Read() */

	IN_IGNORED    = uint32(syscall.IN_IGNORED)    // Watch was removed explicitly or automatically
	IN_ISDIR      = uint32(syscall.IN_ISDIR)      // Subject of this event is a directory.
	IN_Q_OVERFLOW = uint32(syscall.IN_Q_OVERFLOW) // Event queue overflowed (wd is -1 for this event).

	IN_UNMOUNT = uint32(syscall.IN_UNMOUNT) // Filesystem containing watched object was unmounted.
)

var in_mapping = map[uint32]string{
	IN_ACCESS:        "IN_ACCESS",
	IN_ATTRIB:        "IN_ATTRIB",
	IN_CLOSE_WRITE:   "IN_CLOSE_WRITE",
	IN_CLOSE_NOWRITE: "IN_CLOSE_NOWRITE",
	IN_CREATE:        "IN_CREATE",
	IN_DELETE:        "IN_DELETE",
	IN_DELETE_SELF:   "IN_DELETE_SELF",
	IN_MODIFY:        "IN_MODIFY",
	IN_MOVE_SELF:     "IN_MOVE_SELF",
	IN_MOVED_FROM:    "IN_MOVED_FROM",
	IN_MOVED_TO:      "IN_MOVED_TO",
	IN_OPEN:          "IN_OPEN",
	IN_IGNORED:       "IN_IGNORED",
	IN_ISDIR:         "IN_ISDIR",
	IN_Q_OVERFLOW:    "IN_Q_OVERFLOW",
	IN_UNMOUNT:       "IN_UNMOUNT",
}

func InMaskToString(in_mask uint32) string {
	sb := &strings.Builder{}
	divide := false
	for mask, str := range in_mapping {
		if in_mask&mask == mask {
			if divide {
				sb.WriteString("|")
			}
			sb.WriteString(str)
			divide = true
		}
	}
	return sb.String()
}

// InotifyEvent is the go representation of inotify_event found in sys/inotify.h
type InotifyEvent struct {
	// Watch descriptor
	Wd uint32
	// File or directory name
	Name string
	// Contains bits that describe the event that occurred
	Mask uint32
	// Usually 0, but if events (like IN_MOVED_FROM and IN_MOVED_TO) are linked then they will have equal cookie
	Cookie uint32
}

func (i InotifyEvent) GoString() string {
	return fmt.Sprintf("gonotify.InotifyEvent{Wd=%#v, Name=%s, Cookie=%#v, Mask=%#v=%s", i.Wd, i.Name, i.Cookie, i.Mask, InMaskToString(i.Mask))
}

func (i InotifyEvent) String() string {
	return fmt.Sprintf("{Wd=%d, Name=%s, Cookie=%d, Mask=%s", i.Wd, i.Name, i.Cookie, InMaskToString(i.Mask))
}

// FileEvent is the wrapper around InotifyEvent with additional Eof marker. Reading from
// FileEvents from DirWatcher.C or FileWatcher.C may end with Eof when underlying inotify is closed
type FileEvent struct {
	InotifyEvent
	Eof bool
}
