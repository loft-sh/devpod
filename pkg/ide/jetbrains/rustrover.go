package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	RustRoverProductCode           = "RR"
	RustRoverDownloadAmd64Template = "https://download.jetbrains.com/rust/rustrover-%s.tar.gz"
	RustRoverDownloadArm64Template = "https://download.jetbrains.com/rust/rustrover-%s-aarch64.tar.gz"
)

var RustRoverOptions = ide.Options{
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

func NewRustRoverServer(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(RustRoverOptions, values, RustRoverProductCode, RustRoverDownloadAmd64Template, RustRoverDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "rustrover",
		DisplayName:   "RustRover",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
