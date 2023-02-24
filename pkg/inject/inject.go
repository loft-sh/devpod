package inject

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/pkg/errors"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
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
	downloadAmd64,
	downloadArm64 string,
	preferDownload,
	chmodPath bool,
	command string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	timeout time.Duration,
	log log.Logger,
) error {
	// generate script
	t, err := template.FillTemplate(Script, map[string]string{
		"Command":         command,
		"ExistsCheck":     existsCheck,
		"InstallDir":      path.Dir(remotePath),
		"InstallFilename": path.Base(remotePath),
		"PreferDownload":  strconv.FormatBool(preferDownload),
		"ChmodPath":       strconv.FormatBool(chmodPath),
		"DownloadAmd":     downloadAmd64,
		"DownloadArm":     downloadArm64,
	})
	if err != nil {
		return err
	}
	log.Debugf("execute inject script")
	defer log.Debugf("done injecting")

	// start script
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdinWriter.Close()

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()

	// check if context is done
	select {
	case <-ctx.Done():
		return context.Canceled
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

		err := exec(cancelCtx, t, stdinReader, stdoutWriter, stderr)
		if err != nil && err != context.Canceled && !strings.Contains(err.Error(), "signal: ") {
			execErrChan <- err
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

		wasExecuted, err := inject(localFile, stdoutReader, stdout, stdinWriter, stdin, timeout, log)
		injectChan <- injectResult{
			wasExecuted: wasExecuted,
			err:         err,
		}
	}()

	// wait here
	var result injectResult
	select {
	case err = <-execErrChan:
		result = <-injectChan
	case result = <-injectChan:
	}

	// prefer result error
	if result.err != nil {
		return result.err
	} else if err != nil {
		return err
	} else if result.wasExecuted || command == "" {
		return nil
	}

	return exec(ctx, command, stdin, stdout, stderr)
}

func inject(localFile LocalFile, stdout io.ReadCloser, stdoutOut io.Writer, stdin io.WriteCloser, stdinOut io.Reader, timeout time.Duration, log log.Logger) (bool, error) {
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
		return false, errors.Wrap(err, "write to stdin")
	}

	// wait until we read something
	line, err = readLine(stdout)
	if err != nil {
		return false, err
	}

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
			return false, fmt.Errorf("unexpected line during inject: %s", string(line))
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
	return true, pipe(
		stdoutOut, stdout,
		stdin, stdinOut,
	)
}

func readLine(reader io.Reader) (string, error) {
	// we always only read a single byte
	buf := make([]byte, 1, 1)
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

func pipe(toStdout io.Writer, fromStdout io.Reader, toStdin io.Writer, fromStdin io.Reader) error {
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
