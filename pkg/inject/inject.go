package inject

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
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
	execErrChan := make(chan error, 1)
	go func() {
		defer stdoutWriter.Close()
		defer log.Debugf("done exec")

		err := exec(cancelCtx, scriptRawCode, stdinReader, stdoutWriter, delayedStderr)
		if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "signal: ") {
			execErrChan <- command.WrapCommandError(delayedStderr.Buffer(), err)
		} else {
			execErrChan <- nil
		}
	}()

	// inject file
	injectChan := make(chan injectResult, 1)
	go func() {
		defer stdinWriter.Close()
		defer log.Debugf("done inject")

		wasExecuted, err := inject(localFile, stdinWriter, stdin, stdoutReader, stdout, delayedStderr, timeout, log)
		injectChan <- injectResult{
			wasExecuted: wasExecuted,
			err:         command.WrapCommandError(delayedStderr.Buffer(), err),
		}
	}()

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

func inject(
	localFile LocalFile,
	stdin io.WriteCloser,
	stdinOut io.Reader,
	stdout io.ReadCloser,
	stdoutOut io.Writer,
	delayedStderr *delayedWriter,
	timeout time.Duration,
	log log.Logger,
) (bool, error) {
	// wait until we read start
	var line string
	errChan := make(chan error)
	go func() {
		var err error
		line, err = readLine(stdout)
		errChan <- err
	}()

	// wait for line to be read
	err := waitForMessage(errChan, timeout)
	if err != nil {
		return false, err
	}

	err = performMutualHandshake(line, stdin)
	if err != nil {
		return false, err
	}

	// wait until we read something
	line, err = readLine(stdout)
	if err != nil {
		return false, err
	}
	log.Debugf("Received line after pong: %v", line)

	lineStr := strings.TrimSpace(line)
	if isInjectingOfBinaryNeeded(lineStr) {
		log.Debugf("Inject binary")
		defer log.Debugf("Done injecting binary")

		fileReader, err := getFileReader(localFile, lineStr)
		if err != nil {
			return false, err
		}
		defer fileReader.Close()
		err = injectBinary(fileReader, stdin, stdout)
		if err != nil {
			return false, err
		}
		_ = stdout.Close()
		// start exec with command
		return false, nil
	} else if lineStr != "done" {
		return false, fmt.Errorf("unexpected message during inject: %s", lineStr)
	}

	if stdoutOut == nil {
		stdoutOut = io.Discard
	}
	if stdinOut == nil {
		stdinOut = bytes.NewReader(nil)
	}

	// now pipe reader into stdout
	delayedStderr.Start()
	return true, pipe(
		stdin, stdinOut,
		stdoutOut, stdout,
	)
}

func isInjectingOfBinaryNeeded(lineStr string) bool {
	return strings.HasPrefix(lineStr, "ARM-")
}

func getFileReader(localFile LocalFile, lineStr string) (io.ReadCloser, error) {
	isArm := strings.TrimPrefix(lineStr, "ARM-") == "true"
	return localFile(isArm)
}

func performMutualHandshake(line string, stdin io.WriteCloser) error {
	// check for string
	if strings.TrimSpace(line) != "ping" {
		return fmt.Errorf("unexpected start line: %v", line)
	}

	// send our response
	_, err := stdin.Write([]byte("pong\n"))
	if err != nil {
		return perrors.Wrap(err, "write to stdin")
	}

	// successful handshake
	return nil
}

func injectBinary(
	fileReader io.ReadCloser,
	stdin io.WriteCloser,
	stdout io.ReadCloser,
) error {
	// copy into writer
	_, err := io.Copy(stdin, fileReader)
	if err != nil {
		return err
	}

	// close stdin
	_ = stdin.Close()

	// wait for done
	line, err := readLine(stdout)
	if err != nil {
		return err
	} else if strings.TrimSpace(line) != "done" {
		return fmt.Errorf("unexpected line during inject: %s", line)
	}
	return nil
}

func waitForMessage(errChannel chan error, timeout time.Duration) error {
	select {
	case err := <-errChannel:
		return err
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

func readLine(reader io.Reader) (string, error) {
	// we always only read a single byte
	buf := make([]byte, 1)
	str := ""
	for {
		n, err := reader.Read(buf)
		if err != nil {
			return "", err
		} else if n == 0 {
			continue
		} else if buf[0] == '\n' {
			return str, nil
		}

		str += string(buf)
	}
}

func pipe(toStdin io.Writer, fromStdin io.Reader, toStdout io.Writer, fromStdout io.Reader) error {
	errChan := make(chan error, 2)
	go func() {
		_, err := io.Copy(toStdout, fromStdout)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(toStdin, fromStdin)
		errChan <- err
	}()
	return <-errChan
}
