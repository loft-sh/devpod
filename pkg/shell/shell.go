package shell

import (
	"context"
	command2 "github.com/loft-sh/devpod/pkg/command"
	"github.com/pkg/errors"
	"io"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ExecuteCommandWithShell(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, environ []string) error {
	// try to find a proper shell
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

	// run emulated shell
	return RunEmulatedShell(ctx, command, stdin, stdout, stderr, environ)
}

func RunEmulatedShell(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, env []string) error {
	// Let's parse the complete command
	parsed, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return errors.Wrap(err, "parse shell command")
	}

	// Get current working directory
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	// create options
	options := []interp.RunnerOption{
		interp.StdIO(stdin, stdout, stderr),
		interp.Env(expand.ListEnviron(env...)),
		interp.Dir(dir),
		interp.ExecHandler(interp.DefaultExecHandler(2 * time.Second)),
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
