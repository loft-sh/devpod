package copy

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

func Chown(path string, userName string) error {
	if userName == "" {
		return nil
	}

	userId, err := user.Lookup(userName)
	if err != nil {
		return errors.Wrap(err, "lookup user")
	}

	uid, _ := strconv.Atoi(userId.Uid)
	return os.Chown(path, uid, -1)
}

func ChownR(path string, userName string) error {
	if userName == "" {
		return nil
	}

	userId, err := user.Lookup(userName)
	if err != nil {
		return errors.Wrap(err, "lookup user")
	}

	uid, _ := strconv.Atoi(userId.Uid)
	return filepath.WalkDir(path, func(name string, dirEntry fs.DirEntry, err error) error {
		info, err := dirEntry.Info()
		if err != nil {
			return nil
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if ok && stat.Uid == uint32(uid) {
			return nil
		}

		if err == nil {
			err = os.Chown(name, uid, -1)
		}
		return err
	})
}

func Directory(scrDir, dest string) error {
	entries, err := os.ReadDir(scrDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := CreateIfNotExists(destPath, 0755); err != nil {
				return err
			}
			if err := Directory(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			if err := Symlink(sourcePath, destPath); err != nil {
				return err
			}
		default:
			if err := File(sourcePath, destPath, 0666); err != nil {
				return err
			}
		}

		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}

		fInfo, err := entry.Info()
		if err != nil {
			return err
		}

		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, fInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func File(srcFile, dstFile string, perm os.FileMode) error {
	out, err := os.OpenFile(dstFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	defer in.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func Symlink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}
