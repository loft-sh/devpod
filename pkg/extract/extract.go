package extract

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"time"

	perrors "github.com/pkg/errors"
)

type Options struct {
	StripLevels int

	Perm *os.FileMode
	UID  *int
	GID  *int
}

type Option func(o *Options)

func StripLevels(levels int) Option {
	return func(o *Options) {
		o.StripLevels = levels
	}
}

func Extract(origReader io.Reader, destFolder string, options ...Option) error {
	extractOptions := &Options{}
	for _, o := range options {
		o(extractOptions)
	}

	// read ahead
	bufioReader := bufio.NewReaderSize(origReader, 1024*1024)
	testBytes, err := bufioReader.Peek(2) // read 2 bytes
	if err != nil {
		return err
	}

	// is gzipped?
	var reader io.Reader
	if testBytes[0] == 31 && testBytes[1] == 139 {
		gzipReader, err := gzip.NewReader(bufioReader)
		if err != nil {
			return perrors.Errorf("error decompressing: %v", err)
		}
		defer gzipReader.Close()

		reader = gzipReader
	} else {
		reader = bufioReader
	}

	tarReader := tar.NewReader(reader)
	for {
		shouldContinue, err := extractNext(tarReader, destFolder, extractOptions)
		if err != nil {
			return perrors.Wrap(err, "decompress")
		} else if !shouldContinue {
			return nil
		}
	}
}

func extractNext(tarReader *tar.Reader, destFolder string, options *Options) (bool, error) {
	header, err := tarReader.Next()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return false, perrors.Wrap(err, "tar reader next")
		}

		return false, nil
	}

	relativePath := getRelativeFromFullPath("/"+header.Name, "")
	if options.StripLevels > 0 {
		for i := 0; i < options.StripLevels; i++ {
			relativePath = strings.TrimPrefix(relativePath, "/")
			index := strings.Index(relativePath, "/")
			if index == -1 {
				break
			}

			relativePath = relativePath[index+1:]
		}

		relativePath = "/" + relativePath
	}
	outFileName := path.Join(destFolder, relativePath)
	baseName := path.Dir(outFileName)

	dirPerm := os.ModePerm
	if options.Perm != nil {
		dirPerm = *options.Perm
	}

	// Check if newer file is there and then don't override?
	if err := os.MkdirAll(baseName, dirPerm); err != nil {
		return false, err
	}

	// whats the file perm?
	filePerm := os.FileMode(0644)
	if options.Perm != nil {
		filePerm = *options.Perm
	}

	// Is dir?
	if header.Typeflag == tar.TypeDir {
		if err := os.MkdirAll(outFileName, dirPerm); err != nil {
			return false, err
		}

		return true, nil
	} else if header.Typeflag == tar.TypeSymlink {
		err := os.Symlink(header.Linkname, outFileName)
		if err != nil {
			return false, err
		}

		return true, nil
	} else if header.Typeflag == tar.TypeLink {
		err := os.Link(header.Linkname, outFileName)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	// Create / Override file
	outFile, err := os.OpenFile(outFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		// Try again after 5 seconds
		time.Sleep(time.Second * 5)
		outFile, err = os.OpenFile(outFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm)
		if err != nil {
			return false, perrors.Wrapf(err, "create %s", outFileName)
		}
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, tarReader); err != nil {
		return false, perrors.Wrapf(err, "io copy tar reader %s", outFileName)
	}
	if err := outFile.Close(); err != nil {
		return false, perrors.Wrapf(err, "out file close %s", outFileName)
	}

	// Set permissions
	if options.Perm == nil {
		_ = os.Chmod(outFileName, header.FileInfo().Mode()|0600)
	}

	// Set mod time from tar header
	_ = os.Chtimes(outFileName, time.Now(), header.FileInfo().ModTime())

	return true, nil
}

func getRelativeFromFullPath(fullpath string, prefix string) string {
	return strings.TrimPrefix(strings.ReplaceAll(strings.ReplaceAll(fullpath[len(prefix):], "\\", "/"), "//", "/"), ".")
}
