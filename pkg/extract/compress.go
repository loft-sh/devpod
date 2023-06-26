package extract

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"
)

func WriteTar(writer io.Writer, localPath string, compress bool) error {
	absolute, err := filepath.Abs(localPath)
	if err != nil {
		return errors.Wrap(err, "absolute")
	}

	// Check if target is there
	stat, err := os.Stat(absolute)
	if err != nil {
		return errors.Wrap(err, "stat")
	}

	// Use compression
	gw := writer
	if compress {
		gwWriter := gzip.NewWriter(writer)
		defer gwWriter.Close()

		gw = gwWriter
	}

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	// When its a file we copy the file to the toplevel of the tar
	if !stat.IsDir() {
		return NewArchiver(filepath.Dir(absolute), tarWriter).AddToArchive(filepath.Base(absolute))
	}

	// When its a folder we copy the contents and not the folder itself to the
	// toplevel of the tar
	return NewArchiver(absolute, tarWriter).AddToArchive("")
}

// Archiver is responsible for compressing specific files and folders within a target directory
type Archiver struct {
	basePath     string
	writer       *tar.Writer
	writtenFiles map[string]bool
}

// NewArchiver creates a new archiver
func NewArchiver(basePath string, writer *tar.Writer) *Archiver {
	return &Archiver{
		basePath:     basePath,
		writer:       writer,
		writtenFiles: map[string]bool{},
	}
}

// AddToArchive adds a new path to the archive
func (a *Archiver) AddToArchive(relativePath string) error {
	absFilepath := path.Join(a.basePath, relativePath)
	if a.writtenFiles[relativePath] {
		return nil
	}

	// We skip files that are suddenly not there anymore
	stat, err := os.Lstat(absFilepath)
	if err != nil {
		// config.Logf("[Upstream] Couldn't stat file %s: %s\n", absFilepath, err.Error())
		return nil
	}

	if stat.IsDir() {
		// Recursively tar folder
		return a.tarFolder(relativePath, stat)
	}

	return a.tarFile(relativePath, stat)
}

func (a *Archiver) tarFolder(target string, targetStat os.FileInfo) error {
	filePath := path.Join(a.basePath, target)
	files, err := os.ReadDir(filePath)
	if err != nil {
		// config.Logf("[Upstream] Couldn't read dir %s: %s\n", filepath, err.Error())
		return nil
	}

	if len(files) == 0 && target != "" {
		// Case empty directory
		hdr, _ := tar.FileInfoHeader(targetStat, filePath)
		hdr.Uid = 0
		hdr.Gid = 0
		hdr.Mode = fillGo18FileTypeBits(int64(chmodTarEntry(os.FileMode(hdr.Mode))), targetStat)
		hdr.Name = target
		if err := a.writer.WriteHeader(hdr); err != nil {
			return errors.Wrap(err, "tar write header")
		}
		a.writtenFiles[target] = true
	}

	for _, dirEntry := range files {
		f, err := dirEntry.Info()
		if err != nil {
			continue
		}

		if err = a.AddToArchive(path.Join(target, f.Name())); err != nil {
			return errors.Wrap(err, "recursive tar "+f.Name())
		}
	}

	return nil
}

func (a *Archiver) tarFile(target string, targetStat os.FileInfo) error {
	var err error
	filepath := path.Join(a.basePath, target)

	// don't resolve symlinks
	linkName := ""
	if targetStat.Mode()&os.ModeSymlink == os.ModeSymlink {
		linkName, err = os.Readlink(filepath)
		if err != nil {
			return nil
		}
	}

	hdr, err := tar.FileInfoHeader(targetStat, linkName)
	if err != nil {
		return errors.Wrap(err, "create tar file info header")
	}
	hdr.Name = target
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Mode = fillGo18FileTypeBits(int64(chmodTarEntry(os.FileMode(hdr.Mode))), targetStat)
	hdr.ModTime = time.Unix(targetStat.ModTime().Unix(), 0)

	if err := a.writer.WriteHeader(hdr); err != nil {
		return errors.Wrap(err, "tar write header")
	}

	// nothing more to do for non-regular
	if !targetStat.Mode().IsRegular() {
		a.writtenFiles[target] = true
		return nil
	}

	// Case regular file
	f, err := os.Open(filepath)
	if err != nil {
		// We ignore open file and just treat it as okay
		return nil
	}
	defer f.Close()
	copied, err := io.CopyN(a.writer, f, targetStat.Size())
	if err != nil {
		return errors.Wrap(err, "tar copy file")
	} else if copied != targetStat.Size() {
		return errors.New("tar: file truncated during read")
	}

	a.writtenFiles[target] = true
	return nil
}

const (
	modeISDIR  = 040000  // Directory
	modeISFIFO = 010000  // FIFO
	modeISREG  = 0100000 // Regular file
	modeISLNK  = 0120000 // Symbolic link
	modeISBLK  = 060000  // Block special file
	modeISCHR  = 020000  // Character special file
	modeISSOCK = 0140000 // Socket
)

// chmodTarEntry is used to adjust the file permissions used in tar header based
// on the platform the archival is done.
func chmodTarEntry(perm os.FileMode) os.FileMode {
	if runtime.GOOS != "windows" {
		return perm
	}

	// perm &= 0755 // this 0-ed out tar flags (like link, regular file, directory marker etc.)
	permPart := perm & os.ModePerm
	noPermPart := perm &^ os.ModePerm
	// Add the x bit: make everything +x from windows
	permPart |= 0111
	permPart &= 0755

	return noPermPart | permPart
}

// fillGo18FileTypeBits fills type bits which have been removed on Go 1.9 archive/tar
// https://github.com/golang/go/commit/66b5a2f
func fillGo18FileTypeBits(mode int64, fi os.FileInfo) int64 {
	fm := fi.Mode()
	switch {
	case fm.IsRegular():
		mode |= modeISREG
	case fi.IsDir():
		mode |= modeISDIR
	case fm&os.ModeSymlink != 0:
		mode |= modeISLNK
	case fm&os.ModeDevice != 0:
		if fm&os.ModeCharDevice != 0 {
			mode |= modeISCHR
		} else {
			mode |= modeISBLK
		}
	case fm&os.ModeNamedPipe != 0:
		mode |= modeISFIFO
	case fm&os.ModeSocket != 0:
		mode |= modeISSOCK
	}
	return mode
}
