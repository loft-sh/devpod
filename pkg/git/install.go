package git

import (
	"fmt"
	"os/exec"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func InstallBinary(log log.Logger) error {
	writer := log.Writer(logrus.InfoLevel, false)
	errwriter := log.Writer(logrus.ErrorLevel, false)
	defer writer.Close()
	defer errwriter.Close()

	// try to install git via apt / apk
	if !command.Exists("apt") && !command.Exists("apk") {
		// TODO: use golang git implementation
		return fmt.Errorf("couldn't find a package manager to install git")
	}

	if command.Exists("apt") {
		log.Infof("Git command is missing, try to install git with apt...")
		cmd := exec.Command("apt", "update")
		cmd.Stdout = writer
		cmd.Stderr = errwriter
		err := cmd.Run()
		if err != nil {
			return errors.Wrap(err, "run apt update")
		}
		cmd = exec.Command("apt", "-y", "install", "git")
		cmd.Stdout = writer
		cmd.Stderr = errwriter
		err = cmd.Run()
		if err != nil {
			return errors.Wrap(err, "run apt install git -y")
		}
	} else if command.Exists("apk") {
		log.Infof("Git command is missing, try to install git with apk...")
		cmd := exec.Command("apk", "update")
		cmd.Stdout = writer
		cmd.Stderr = errwriter
		err := cmd.Run()
		if err != nil {
			return errors.Wrap(err, "run apk update")
		}
		cmd = exec.Command("apk", "add", "git")
		cmd.Stdout = writer
		cmd.Stderr = errwriter
		err = cmd.Run()
		if err != nil {
			return errors.Wrap(err, "run apk add git")
		}
	}

	// is git available now?
	if !command.Exists("git") {
		return fmt.Errorf("couldn't install git")
	}

	log.Donef("Successfully installed git")

	return nil
}
