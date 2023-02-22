package inject

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"github.com/loft-sh/devpod/pkg/template"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

//go:embed inject.sh
var Script string

type ExecFunc func(command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

type LocalFile func(arm bool) (io.ReadCloser, error)

func Inject(exec ExecFunc, localFile LocalFile, existsCheck string, remotePath, downloadAmd64, downloadArm64 string, preferDownload, chmodPath bool, timeout time.Duration) error {
	// generate script
	t, err := template.FillTemplate(Script, map[string]string{
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

	// start script
	stderr := &bytes.Buffer{}
	errChan := make(chan error, 2)
	go func() {
		errChan <- exec(t, stdinReader, stdoutWriter, stderr)
	}()

	// inject file
	go func() {
		defer stdoutWriter.Close()
		defer stdinWriter.Close()

		errChan <- inject(localFile, stdoutReader, stdinWriter, timeout)
	}()

	return <-errChan
}

func inject(localFile LocalFile, stdout io.Reader, stdin io.WriteCloser, timeout time.Duration) error {
	// wait until we read start
	reader := bufio.NewReader(stdout)
	var line []byte
	errChan := make(chan error)
	go func() {
		var err error
		line, err = reader.ReadBytes('\n')
		errChan <- err
	}()

	// wait for line to be read
	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}

	// check for string
	if strings.TrimSpace(string(line)) != "start" {
		return fmt.Errorf("unexpected start line: %v", string(line))
	}

	// wait until we read something
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return err
	}

	// check if we need to inject the file
	lineStr := strings.TrimSpace(string(line))
	if strings.HasPrefix(lineStr, "ARM-") {
		isArm := strings.TrimPrefix(lineStr, "ARM-") == "true"
		reader, err := localFile(isArm)
		if err != nil {
			return err
		}
		defer reader.Close()

		// copy into writer
		_, err = io.Copy(stdin, reader)
		if err != nil {
			return err
		}

		// close stdin
		_ = stdin.Close()
	} else if lineStr == "done" {
		return nil
	}

	// wait for done
	line, err = reader.ReadBytes('\n')
	if err != nil {
		return err
	} else if strings.TrimSpace(string(line)) != "done" {
		return fmt.Errorf("unexpected line during inject: %s", string(line))
	}

	return nil
}
