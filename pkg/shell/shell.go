package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	command2 "github.com/loft-sh/devpod/pkg/command"
	"github.com/pkg/errors"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func ExecuteCommandWithShell(
	ctx context.Context,
	command string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	environ []string,
) error {
	command = strings.ReplaceAll(command, "\r", "")

	// try to find a proper shell
	if runtime.GOOS != "windows" {
		if command2.Exists("bash") {
			cmd := exec.CommandContext(ctx, "bash", "-c", command)
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			cmd.Env = environ
			return cmd.Run()
		} else if command2.Exists("sh") {
			cmd := exec.CommandContext(ctx, "sh", "-c", command)
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			cmd.Env = environ
			return cmd.Run()
		}
	}

	// run emulated shell
	return RunEmulatedShell(ctx, command, stdin, stdout, stderr, environ)
}

func RunEmulatedShell(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, env []string) error {
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
