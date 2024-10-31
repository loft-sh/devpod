package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	DataSpellProductCode           = "DS"
	DataSpellDownloadAmd64Template = "https://download.jetbrains.com/ds/dataspell-%s.tar.gz"
	DataSpellDownloadArm64Template = "https://download.jetbrains.com/ds/dataspell-%s-aarch64.tar.gz"
)

var DataSpellOptions = ide.Options{
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

func NewDataSpellServer(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(DataSpellOptions, values, DataSpellProductCode, DataSpellDownloadAmd64Template, DataSpellDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "dataspell",
		DisplayName:   "DataSpell",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
