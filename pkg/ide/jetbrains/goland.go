package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	GolandProductCode           = "GO"
	GolandDownloadAmd64Template = "https://download.jetbrains.com/go/goland-%s.tar.gz"
	GolandDownloadArm64Template = "https://download.jetbrains.com/go/goland-%s-aarch64.tar.gz"
)

var GolandOptions = ide.Options{
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

func NewGolandServer(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(GolandOptions, values, GolandProductCode, GolandDownloadAmd64Template, GolandDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "goland",
		DisplayName:   "Goland",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
