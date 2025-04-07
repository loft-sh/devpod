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
	cmd.Dir = os.TempDir()

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
	cmd.Dir = os.TempDir()

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
	var execOut, execErr bytes.Buffer

	// Helper to run the command with an optional override dir
	run := func(overrideDir string) (string, string, error) {
		execOut.Reset()
		execErr.Reset()

		cmd := exec.CommandContext(ctx, filepath.Join(f.DevpodBinDir, f.DevpodBinName), args...)
		cmd.Stdout = io.MultiWriter(os.Stdout, &execOut)
		cmd.Stderr = io.MultiWriter(os.Stderr, &execErr)
		if overrideDir != "" {
			cmd.Dir = overrideDir
		}

		err := cmd.Run()
		return execOut.String(), execErr.String(), err
	}

	stdout, stderr, err := run("") // try without override
	if err != nil {
		// Check for getwd-related error (some implementations return "no such file or directory" or similar)
		if strings.Contains(err.Error(), "getwd") || strings.Contains(err.Error(), "no such file or directory") {
			return run(os.TempDir()) // retry in a safe location
		}
	}

	return stdout, stderr, err
}
