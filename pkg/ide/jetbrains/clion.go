package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	CLionProductCode           = "CL"
	CLionDownloadAmd64Template = "https://download.jetbrains.com/cpp/CLion-%s.tar.gz"
	CLionDownloadArm64Template = "https://download.jetbrains.com/cpp/CLion-%s-aarch64.tar.gz"
)

var CLionOptions = ide.Options{
	VersionOption: {
		Name:        VersionOption,
		Description: "The version for the binary",
		Default:     "latest",
	},
	DownloadArm64Option: {
		Name:        DownloadArm64Option,
		Description: "The download url for the arm64 server binary",
	},
	DownloadAmd64Option: {
		Name:        DownloadAmd64Option,
		Description: "The download url for the amd64 server binary",
	},
}

func NewCLionServer(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(CLionOptions, values, CLionProductCode, CLionDownloadAmd64Template, CLionDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "clion",
		DisplayName:   "CLion",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
