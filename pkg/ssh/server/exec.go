package server

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/ssh"
	perrors "github.com/pkg/errors"
)

func execNonPTY(sess ssh.Session, cmd *exec.Cmd, log log.Logger) (err error) {
	log.Debugf("Execute SSH server command: %s", strings.Join(cmd.Args, " "))
	// init pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// start the command
	err = cmd.Start()
	if err != nil {
		return perrors.Wrap(err, "start command")
	}

	go func() {
		defer stdin.Close()

		_, err := io.Copy(stdin, sess)
		if err != nil {
			log.Debugf("Error piping stdin: %v", err)
		}
	}()

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		_, err := io.Copy(sess, stdout)
		if err != nil {
			log.Debugf("Error piping stdout: %v", err)
		}
	}()

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		_, err := io.Copy(sess.Stderr(), stderr)
		if err != nil {
			log.Debugf("Error piping stderr: %v", err)
		}
	}()

	waitGroup.Wait()
	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func execPTY(
	sess ssh.Session,
	ptyReq ssh.Pty,
	winCh <-chan ssh.Window,
	cmd *exec.Cmd,
	log log.Logger,
) (err error) {
	log.Debugf("Execute SSH server PTY command: %s", strings.Join(cmd.Args, " "))

	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	f, err := startPTY(cmd)
	if err != nil {
		return perrors.Wrap(err, "start pty")
	}
	defer f.Close()

	go func() {
		for win := range winCh {
			setWinSize(f, win.Width, win.Height)
		}
	}()

	go func() {
		defer f.Close()

		// copy stdin
		_, _ = io.Copy(f, sess)
	}()

	stdoutDoneChan := make(chan struct{})
	go func() {
		defer f.Close()
		defer close(stdoutDoneChan)

		// copy stdout
		_, _ = io.Copy(sess, f)
	}()

	err = cmd.Wait()
	if err != nil {
		return err
	}

	select {
	case <-stdoutDoneChan:
	case <-time.After(time.Second):
	}
	return nil
}
