package framework

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecCommand executes the command string with the devpod test binary
func (f *Framework) ExecCommandOutput(ctx context.Context, args []string) (string, error) {
	var execOut bytes.Buffer

	cmd := exec.CommandContext(ctx, filepath.Join(f.DevpodBinDir, f.DevpodBinName), args...)
	cmd.Stdout = io.MultiWriter(os.Stdout, &execOut)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return execOut.String(), nil
}

// ExecCommandStdout executes the command string with the devpod test binary
func (f *Framework) ExecCommandStdout(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, filepath.Join(f.DevpodBinDir, f.DevpodBinName), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// ExecCommand executes the command string with the devpod test binary
func (f *Framework) ExecCommand(ctx context.Context, captureStdOut, searchForString bool, searchString string, args []string) error {
	var execOut bytes.Buffer

	cmd := exec.CommandContext(ctx, filepath.Join(f.DevpodBinDir, f.DevpodBinName), args...)
	cmd.Stdout = io.MultiWriter(os.Stdout, &execOut)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	if captureStdOut && searchForString {
		if strings.Contains(execOut.String(), searchString) {
			return nil
		}

		return fmt.Errorf("expected to find string %s in output", searchString)
	}

	return nil
}

// ExecCommandCapture executes the command string with the devpod test binary, and returns stdout, stderr, and any error that occurred.
func (f *Framework) ExecCommandCapture(ctx context.Context, args []string) (string, string, error) {
	var execOut bytes.Buffer
	var execErr bytes.Buffer

	cmd := exec.CommandContext(ctx, filepath.Join(f.DevpodBinDir, f.DevpodBinName), args...)
	cmd.Stdout = io.MultiWriter(os.Stdout, &execOut)
	cmd.Stderr = io.MultiWriter(os.Stderr, &execErr)

	err := cmd.Run()
	return execOut.String(), execErr.String(), err
}
