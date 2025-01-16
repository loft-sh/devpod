## Gonotify 

Simple Golang inotify wrapper.

[![Build Status](https://travis-ci.org/illarion/gonotify.svg?branch=master)](https://travis-ci.org/illarion/gonotify)
[![GoDoc](https://godoc.org/github.com/illarion/gonotify?status.svg)](https://godoc.org/github.com/illarion/gonotify)

### Provides following primitives:

* Low level
  * `Inotify` - wrapper around [inotify(7)](http://man7.org/linux/man-pages/man7/inotify.7.html)
  * `InotifyEvent` - generated file/folder event. Contains `Name` (full path), watch descriptior and `Mask` that describes the event.

* Higher level
  * `FileWatcher` - higher level utility, helps to watch the list of files for changes, creation or removal
  * `DirWatcher` - higher level utility, recursively watches given root folder for added, removed or changed files.
  * `FileEvent` - embeds `InotifyEvent` and keeps additional field `Eof` to notify user that there will be no more events.

Use `FileWatcher` and `DirWatcher` as an example and build your own utility classes.

## License
MIT. See LICENSE file for more details.

