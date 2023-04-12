package framework

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ExecCommand executes the command string with the devpod test binary
func (f *Framework) ExecCommand(ctx context.Context, captureStdOut, searchForString bool, searchString string, args []string) error {
	var execErr bytes.Buffer
	var execOut bytes.Buffer

	prout, pwout, err := os.Pipe()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, f.DevpodBinDir+"/"+f.DevpodBinName, args...)
	cmd.Stdout = pwout
	cmd.Stderr = &execErr

	if err := cmd.Start(); err != nil {
		return err
	}

	outReader := io.TeeReader(prout, os.Stdout)
	go io.Copy(&execOut, outReader)

	if err := cmd.Wait(); err != nil {
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
