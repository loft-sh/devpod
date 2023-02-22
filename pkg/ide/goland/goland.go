package goland

import (
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/log"
)

const GolandDownloadAmd64 = "https://download.jetbrains.com/go/goland-2022.3.2.tar.gz"
const GolandDownloadArm64 = "https://download.jetbrains.com/go/goland-2022.3.2-aarch64.tar.gz"

func NewGolandServer(log log.Logger) ide.IDE {
	return &golandServer{
		log: log,
	}
}

type golandServer struct {
	log log.Logger
}

func (o *golandServer) Install() error {

	return nil
}
