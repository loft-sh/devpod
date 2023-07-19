package single

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofrs/flock"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/pkg/errors"
)

type CreateCommand func() (*exec.Cmd, error)

func Single(file string, createCommand CreateCommand) error {
	file = filepath.Join(os.TempDir(), file)
	fileLock := flock.New(file + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return errors.Wrap(err, "acquire lock")
	} else if !locked {
		return nil
	}
	defer func(fileLock *flock.Flock) {
		_ = fileLock.Unlock()
	}(fileLock)

	// check if marker file is there
	pid, err := os.ReadFile(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		// check if process id exists
		isRunning, err := command.IsRunning(string(pid))
		if err != nil {
			return err
		} else if isRunning {
			return nil
		}
	}

	// create command
	cmd, err := createCommand()
	if err != nil {
		return err
	}

	// pipe streams into file.streams
	f, err := os.Create(file + ".streams")
	if err != nil {
		return err
	}
	if cmd.Stderr == nil {
		cmd.Stderr = f
	}
	if cmd.Stdout == nil {
		cmd.Stdout = f
	}

	// start process
	err = cmd.Start()
	if err != nil {
		return err
	}

	// wait until we have a process id
	for cmd.Process.Pid < 0 {
		time.Sleep(time.Millisecond)
	}

	// write pid to file
	err = os.WriteFile(file, []byte(strconv.Itoa(cmd.Process.Pid)), os.ModePerm)
	if err != nil {
		return err
	}

	// release process resources
	err = cmd.Process.Release()
	if err != nil {
		return err
	}

	return nil
}
