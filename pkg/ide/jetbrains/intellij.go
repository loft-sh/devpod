package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	IntellijProductCode           = "IU"
	IntellijDownloadAmd64Template = "https://download.jetbrains.com/idea/ideaIU-%s.tar.gz"
	IntellijDownloadArm64Template = "https://download.jetbrains.com/idea/ideaIU-%s-aarch64.tar.gz"
)

var IntellijOptions = ide.Options{
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

func NewIntellij(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(IntellijOptions, values, IntellijProductCode, IntellijDownloadAmd64Template, IntellijDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "intellij",
		DisplayName:   "Intellij",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
