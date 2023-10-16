package inject

import (
	"context"
	_ "embed"
	"io"
	"os"
	"time"

	"github.com/loft-sh/log"
)

//go:embed inject.sh
var Script string

type ExecFunc func(ctx context.Context, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

type LocalFile func(arm bool) (io.ReadCloser, error)

type injectResult struct {
	wasExecuted bool
	err         error
}

func InjectAndExecute(
	ctx context.Context,
	exec ExecFunc,
	localFile LocalFile,
	scriptParams *Params,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	timeout time.Duration,
	log log.Logger,
) (bool, error) {
	scriptRawCode, err := GenerateScript(Script, scriptParams)
	if err != nil {
		return true, err
	}

	log.Debugf("execute inject script")
	if scriptParams.PreferAgentDownload {
		log.Debugf("download agent from %s", scriptParams.DownloadURLs.Base)
	}

	defer log.Debugf("done injecting")

	// start script
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return true, err
	}
	defer stdinWriter.Close()

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return true, err
	}
	defer stdoutWriter.Close()

	// delayed stderr
	delayedStderr := newDelayedWriter(stderr)

	// check if context is done
	select {
	case <-ctx.Done():
		return true, context.Canceled
	default:
	}

	// create cancel context
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start execution of inject.sh
	execErrChan := runServerSide(cancelCtx, exec, stdinReader, stdinWriter, stdoutWriter, scriptRawCode, delayedStderr, log)
	injectChan := runClientSide(cancel, localFile, stdin, stdout, stdinWriter, stdoutWriter, stdoutReader, delayedStderr, timeout, log)

	// wait here
	var result injectResult
	select {
	case err = <-execErrChan:
		result = <-injectChan
	case result = <-injectChan:
		// we don't wait for the command termination here and will just retry on error
	}

	// prefer result error
	if result.err != nil {
		return result.wasExecuted, result.err
	} else if err != nil {
		return result.wasExecuted, err
	} else if result.wasExecuted || scriptParams.Command == "" {
		return result.wasExecuted, nil
	}

	log.Debugf("Rerun command as binary was injected")
	delayedStderr.Start()
	return true, exec(ctx, scriptParams.Command, stdin, stdout, delayedStderr)
}
