package inject

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	perrors "github.com/pkg/errors"
)

func runClientSide(
	cancel context.CancelFunc,
	localFile LocalFile,
	stdin io.Reader,
	stdout io.Writer,
	stdinWriter io.WriteCloser,
	stdoutWriter io.WriteCloser,
	stdoutReader io.ReadCloser,
	delayedStderr *delayedWriter,
	timeout time.Duration,
	log log.Logger,
) chan injectResult {
	// inject file
	injectChan := make(chan injectResult, 1)
	go func() {
		defer stdoutWriter.Close()
		defer stdinWriter.Close()
		defer log.Debugf("done inject")
		defer cancel()

		wasExecuted, err := inject(localFile, stdinWriter, stdin, stdoutReader, stdout, delayedStderr, timeout, log)
		injectChan <- injectResult{
			wasExecuted: wasExecuted,
			err:         command.WrapCommandError(delayedStderr.Buffer(), err),
		}
	}()

	return injectChan
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

func getFileReader(localFile LocalFile, lineStr string) (io.ReadCloser, error) {
	isArm := strings.TrimPrefix(lineStr, "ARM-") == "true"
	return localFile(isArm)
}

func isInjectingOfBinaryNeeded(lineStr string) bool {
	return strings.HasPrefix(lineStr, "ARM-")
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
