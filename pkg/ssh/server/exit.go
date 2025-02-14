package server

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/ssh"
	perrors "github.com/pkg/errors"
)

func exitWithError(sess ssh.Session, err error, log log.Logger) {
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(perrors.Cause(err), &exitError) {
			log.Errorf("Exit error: %v", err)
			msg := strings.TrimPrefix(err.Error(), "exec: ")
			if _, err := sess.Stderr().Write([]byte(msg)); err != nil {
				log.Errorf("failed to write error to session: %v", err)
			}
		}
	}

	// always exit session
	err = sess.Exit(exitCode(err))
	if err != nil {
		log.Errorf("session failed to exit: %v", err)
	}
}

func exitCode(err error) int {
	err = perrors.Cause(err)
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 1
	}

	return exitErr.ExitCode()
}
