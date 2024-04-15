package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	RiderProductCode           = "RD"
	RiderDownloadAmd64Template = "https://download.jetbrains.com/rider/JetBrains.Rider-%s.tar.gz"
	RiderDownloadArm64Template = "https://download.jetbrains.com/rider/JetBrains.Rider-%s-aarch64.tar.gz"
)

var RiderOptions = ide.Options{
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

func NewRiderServer(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(RiderOptions, values, RiderProductCode, RiderDownloadAmd64Template, RiderDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "rider",
		DisplayName:   "Rider",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
