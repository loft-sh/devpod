package inject

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
)

func runServerSide(
	cancelCtx context.Context,
	exec ExecFunc,
	stdinReader io.ReadCloser,
	stdinWriter io.WriteCloser,
	stdoutWriter io.WriteCloser,
	scriptRawCode string,
	delayedStderr *delayedWriter,
	log log.Logger,
) chan error {
	execErrChan := make(chan error, 1)
	go func() {
		defer stdoutWriter.Close()
		defer stdinWriter.Close()
		defer log.Debugf("done exec")

		err := exec(cancelCtx, scriptRawCode, stdinReader, stdoutWriter, delayedStderr)
		if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "signal: ") {
			execErrChan <- command.WrapCommandError(delayedStderr.Buffer(), err)
		} else {
			execErrChan <- nil
		}
	}()

	return execErrChan
}
