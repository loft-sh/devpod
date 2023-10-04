package inject

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/template"
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
	existsCheck string,
	remotePath,
	downloadBase,
	downloadAmd64,
	downloadArm64 string,
	preferDownload,
	chmodPath bool,
	commandToExecute string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	timeout time.Duration,
	log log.Logger,
) (bool, error) {
	// generate script
	t, err := template.FillTemplate(Script, map[string]string{
		"Command":         commandToExecute,
		"ExistsCheck":     existsCheck,
		"InstallDir":      path.Dir(remotePath),
		"InstallFilename": path.Base(remotePath),
		"PreferDownload":  strconv.FormatBool(preferDownload),
		"ChmodPath":       strconv.FormatBool(chmodPath),
		"DownloadBase":    downloadBase,
		"DownloadAmd":     downloadAmd64,
		"DownloadArm":     downloadArm64,
	})
	if err != nil {
		return true, err
	}

	log.Debugf("execute inject script")
	if preferDownload {
		log.Debugf("download agent from %s", downloadBase)
	}

	defer log.Debugf("done injecting")

	t = strings.ReplaceAll(t, "\r", "")

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

	// start script
	execErrChan := make(chan error, 1)
	go func() {
		defer stdoutWriter.Close()
		defer stdinWriter.Close()
		defer log.Debugf("done exec")

		err := exec(cancelCtx, t, stdinReader, stdoutWriter, delayedStderr)
		if err != nil && !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "signal: ") {
			execErrChan <- command.WrapCommandError(delayedStderr.Buffer(), err)
		} else {
			execErrChan <- nil
		}
	}()

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
	} else if result.wasExecuted || commandToExecute == "" {
		return result.wasExecuted, nil
	}

	log.Debugf("Rerun command as binary was injected")
	delayedStderr.Start()
	return true, exec(ctx, commandToExecute, stdin, stdout, delayedStderr)
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
	select {
	case err := <-errChan:
		if err != nil {
			return false, err
		}
	case <-time.After(timeout):
		return false, context.DeadlineExceeded
	}

	// check for string
	if strings.TrimSpace(line) != "ping" {
		return false, fmt.Errorf("unexpected start line: %v", line)
	}

	// send our response
	_, err := stdin.Write([]byte("pong\n"))
	if err != nil {
		return false, perrors.Wrap(err, "write to stdin")
	}

	// wait until we read something
	line, err = readLine(stdout)
	if err != nil {
		return false, err
	}
	log.Debugf("Received line after pong: %v", line)

	// check if we need to inject the file
	lineStr := strings.TrimSpace(line)
	if strings.HasPrefix(lineStr, "ARM-") {
		log.Debugf("Inject binary")
		defer log.Debugf("Done injecting binary")

		isArm := strings.TrimPrefix(lineStr, "ARM-") == "true"
		fileReader, err := localFile(isArm)
		if err != nil {
			return false, err
		}
		defer fileReader.Close()

		// copy into writer
		_, err = io.Copy(stdin, fileReader)
		if err != nil {
			return false, err
		}

		// close stdin
		_ = stdin.Close()

		// wait for done
		line, err = readLine(stdout)
		if err != nil {
			return false, err
		} else if strings.TrimSpace(line) != "done" {
			return false, fmt.Errorf("unexpected line during inject: %s", line)
		}

		// close stdout
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
