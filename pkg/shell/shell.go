package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func RunEmulatedShell(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, env []string) error {
	command = strings.ReplaceAll(command, "\r", "")

	// Let's parse the complete command
	parsed, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return errors.Wrap(err, "parse shell command")
	}

	// use system default as environ if unspecified
	if env == nil {
		env = []string{}
		env = append(env, os.Environ()...)
	}

	// Get current working directory
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	// create options
	defaultOpenHandler := interp.DefaultOpenHandler()
	defaultExecHandler := interp.DefaultExecHandler(2 * time.Second)
	options := []interp.RunnerOption{
		interp.StdIO(stdin, stdout, stderr),
		interp.Env(expand.ListEnviron(env...)),
		interp.Dir(dir),
		interp.ExecHandler(func(ctx context.Context, args []string) error {
			return defaultExecHandler(ctx, args)
		}),
		interp.OpenHandler(func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
			if path == "/dev/null" {
				return devNull{}, nil
			}

			return defaultOpenHandler(ctx, path, flag, perm)
		}),
	}

	// Create shell runner
	r, err := interp.New(options...)
	if err != nil {
		return errors.Wrap(err, "create shell runner")
	}

	// Run command
	err = r.Run(ctx, parsed)
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok && status == 0 {
			return nil
		}

		return err
	}

	return nil
}

var _ io.ReadWriteCloser = devNull{}

type devNull struct{}

func (devNull) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (devNull) Write(p []byte) (int, error) {
	return len(p), nil
}

func (devNull) Close() error {
	return nil
}

func GetShell(userName string) ([]string, error) {
	// try to get a shell
	if runtime.GOOS != "windows" {
		// infere login shell from getent
		shell, err := getUserShell(userName)
		if err == nil {
			return []string{shell}, nil
		}

		// fallback to $SHELL env var
		shell, ok := os.LookupEnv("SHELL")
		if ok {
			return []string{shell}, nil
		}

		// fallback to path discovery
		_, err = exec.LookPath("bash")
		if err == nil {
			return []string{"bash"}, nil
		}

		_, err = exec.LookPath("sh")
		if err == nil {
			return []string{"sh"}, nil
		}
	}

	// fallback to our in-built shell
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}

	return []string{executable, "helper", "sh"}, nil
}

func getUserShell(userName string) (string, error) {
	currentUser, err := findUser(userName)
	if err != nil {
		return "", err
	}
	output, err := exec.Command("getent", "passwd", currentUser.Username).Output()
	if err != nil {
		return "", err
	}

	shell := strings.Split(string(output), ":")
	if len(shell) != 7 {
		return "", fmt.Errorf("unexpected getent format: %s", string(output))
	}

	loginShell := strings.TrimSpace(filepath.Base(shell[6]))
	if loginShell == "nologin" {
		return "", fmt.Errorf("no login shell configured")
	}

	return loginShell, nil
}

func findUser(userName string) (*user.User, error) {
	if userName != "" {
		u, err := user.Lookup(userName)
		if err != nil {
			return nil, err
		}
		return u, nil
	}

	return user.Current()
}
