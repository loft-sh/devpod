package server

import (
	"errors"
	"io"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/loft-sh/ssh"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
)

func sftpHandler(sess ssh.Session, currentUser string, log log.Logger) {
	writer := log.Writer(logrus.DebugLevel, false)
	defer writer.Close()

	user := sess.User()
	if user == currentUser {
		user = ""
	}

	workingDir, _ := command.GetHome(user)
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(writer),
		sftp.WithServerWorkingDirectory(workingDir),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Debugf("sftp server init error: %s\n", err)
		return
	}
	defer server.Close()

	// serve
	err = server.Serve()
	if errors.Is(err, io.EOF) {
		_ = sess.Exit(0)
		return
	}

	if err != nil {
		log.Debugf("sftp server completed with error: %v", err)
	}
	_ = sess.Exit(1)
}
