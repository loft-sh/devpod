package copy

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

func Chown(path string, userName string) error {
	if userName == "" {
		return nil
	}

	userID, err := user.Lookup(userName)
	if err != nil {
		return errors.Wrap(err, "lookup user")
	}

	uid, _ := strconv.Atoi(userID.Uid)
	return os.Lchown(path, uid, -1)
}

func ChownR(path string, userName string) error {
	if userName == "" {
		return nil
	}

	userID, err := user.Lookup(userName)
	if err != nil {
		return errors.Wrap(err, "lookup user")
	}

	uid, _ := strconv.Atoi(userID.Uid)
	gid, _ := strconv.Atoi(userID.Gid)
	return filepath.WalkDir(path, func(name string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := dirEntry.Info()
		if err != nil {
			return nil
		}

		if IsUID(info, uint32(uid)) {
			return nil
		}

		if err == nil {
			_ = os.Lchown(name, uid, gid)
		}
		return err
	})
}

func RenameDirectory(srcDir, dest string) error {
	err := Directory(srcDir, dest)
	if err != nil {
		return err
	}

	return os.RemoveAll(srcDir)
}

func Directory(scrDir, dest string) error {
	if err := CreateIfNotExists(dest, 0755); err != nil {
		return err
	}

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

		err = Lchown(fileInfo, sourcePath, destPath)
		if err != nil {
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
	if err != nil {
		return err
	}
	defer in.Close()

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
