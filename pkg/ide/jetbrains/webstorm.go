package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/log"
)

const WebStormDownloadAmd64Template = "https://download.jetbrains.com/webstorm/WebStorm-%s.tar.gz"
const WebStormDownloadArm64Template = "https://download.jetbrains.com/webstorm/WebStorm-%s-aarch64.tar.gz"

var WebStormOptions = ide.Options{
	VersionOption: {
		Name:        VersionOption,
		Description: "The version for the binary",
		Default:     "2023.1.1",
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

func NewWebStormServer(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(WebStormOptions, values, WebStormDownloadAmd64Template, WebStormDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "webstorm",
		DisplayName:   "WebStorm",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
