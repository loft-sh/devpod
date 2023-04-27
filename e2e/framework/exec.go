package framework

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ExecCommand executes the command string with the devpod test binary
func (f *Framework) ExecCommand(ctx context.Context, captureStdOut, searchForString bool, searchString string, args []string) error {
	var execOut bytes.Buffer

	cmd := exec.CommandContext(ctx, f.DevpodBinDir+"/"+f.DevpodBinName, args...)
	cmd.Stdout = io.MultiWriter(os.Stdout, &execOut)

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
