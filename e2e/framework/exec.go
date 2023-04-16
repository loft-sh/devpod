package framework

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExecCommand executes the command string with the devpod test binary
func (f *Framework) ExecCommand(ctx context.Context, captureStdOut, searchForString bool, searchString string, args []string) error {
	var execErr bytes.Buffer
	var execOut bytes.Buffer

	cmd := exec.CommandContext(ctx, f.DevpodBinDir+"/"+f.DevpodBinName, args...)
	cmd.Stdout = os.Stdout
	if captureStdOut {
		cmd.Stdout = &execOut
	}
	cmd.Stderr = &execErr
	err := cmd.Run()
	if err != nil && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s: %s", err.Error(), execErr.String())
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		if searchForString && captureStdOut {
			if strings.Contains(execOut.String(), searchString) {
				return nil
			}
		}
		return context.DeadlineExceeded
	}
	return nil
}
