package gonotify

import (
	"os"
	"path/filepath"
)

// DirWatcher recursively watches the given root folder, waiting for file events.
// Events can be masked by providing fileMask. DirWatcher does not generate events for
// folders or subfolders.
type DirWatcher struct {
	stopC chan struct{}
	C     chan FileEvent
}

// NewDirWatcher creates DirWatcher recursively waiting for events in the given root folder and
// emitting FileEvents in channel C, that correspond to fileMask. Folder events are ignored (having IN_ISDIR set to 1)
func NewDirWatcher(fileMask uint32, root string) (*DirWatcher, error) {
	dw := &DirWatcher{
		stopC: make(chan struct{}),
		C:     make(chan FileEvent),
	}

	i, err := NewInotify()
	if err != nil {
		return nil, err
	}

	queue := make([]FileEvent, 0, 100)

	err = filepath.Walk(root, func(path string, f os.FileInfo, err error) error {

		if err != nil {
			return nil
		}

		if !f.IsDir() {

			//fake event for existing files
			queue = append(queue, FileEvent{
				InotifyEvent: InotifyEvent{
					Name: path,
					Mask: IN_CREATE,
				},
			})

			return nil
		}
		return i.AddWatch(path, IN_ALL_EVENTS)
	})

	if err != nil {
		i.Close()
		return nil, err
	}

	events := make(chan FileEvent)

	go func() {
		for _, event := range queue {
			events <- event
		}
		queue = nil

		for {

			raw, err := i.Read()
			if err != nil {
				close(events)
				return
			}

			for _, event := range raw {

				// Skip ignored events queued from removed watchers
				if event.Mask&IN_IGNORED == IN_IGNORED {
					continue
				}

				// Add watch for folders created in watched folders (recursion)
				if event.Mask&(IN_CREATE|IN_ISDIR) == IN_CREATE|IN_ISDIR {

					// After the watch for subfolder is added, it may be already late to detect files
					// created there right after subfolder creation, so we should generate such events
					// ourselves:
					filepath.Walk(event.Name, func(path string, f os.FileInfo, err error) error {
						if err != nil {
							return nil
						}

						if !f.IsDir() {
							// fake event, but there can be duplicates of this event provided by real watcher
							events <- FileEvent{
								InotifyEvent: InotifyEvent{
									Name: path,
									Mask: IN_CREATE,
								},
							}
						}

						return nil
					})

					// Wait for further files to be added
					i.AddWatch(event.Name, IN_ALL_EVENTS)

					continue
				}

				// Remove watch for deleted folders
				if event.Mask&IN_DELETE_SELF == IN_DELETE_SELF {
					i.RmWd(event.Wd)
					continue
				}

				// Skip sub-folder events
				if event.Mask&IN_ISDIR == IN_ISDIR {
					continue
				}

				events <- FileEvent{
					InotifyEvent: event,
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-dw.stopC:
				i.Close()
			case event, ok := <-events:
				if !ok {
					dw.C <- FileEvent{
						Eof: true,
					}
					return
				}

				// Skip events not conforming with provided mask
				if event.Mask&fileMask == 0 {
					continue
				}

				dw.C <- event
			}
		}
	}()

	return dw, nil

}

func (d *DirWatcher) Close() {
	select {
	case d.stopC <- struct{}{}:
	default:
	}
}
