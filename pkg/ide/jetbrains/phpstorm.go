package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	PhpStormProductCode           = "PS"
	PhpStormDownloadAmd64Template = "https://download.jetbrains.com/webide/PhpStorm-%s.tar.gz"
	PhpStormDownloadArm64Template = "https://download.jetbrains.com/webide/PhpStorm-%s-aarch64.tar.gz"
)

var PhpStormOptions = ide.Options{
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

func NewPhpStorm(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(PhpStormOptions, values, PhpStormProductCode, PhpStormDownloadAmd64Template, PhpStormDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "phpstorm",
		DisplayName:   "PhpStorm",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
