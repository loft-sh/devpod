package framework

import (
	"runtime"
)

type Framework struct {
	DevpodBinDir  string
	DevpodBinName string
}

func NewDefaultFramework(path string) *Framework {
	var binName = "devpod-"
	switch runtime.GOOS {
	case "darwin":
		binName = binName + "darwin-"
	case "linux":
		binName = binName + "linux-"
	}

	switch runtime.GOARCH {
	case "amd64":
		binName = binName + "amd64"
	case "arm64":
		binName = binName + "arm64"
	}
	return &Framework{DevpodBinDir: path, DevpodBinName: binName}
}
