package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/log"
)

const (
	RubyMineProductCode           = "RM"
	RubyMineDownloadAmd64Template = "https://download.jetbrains.com/ruby/RubyMine-%s.tar.gz"
	RubyMineDownloadArm64Template = "https://download.jetbrains.com/ruby/RubyMine-%s-aarch64.tar.gz"
)

var RubyMineOptions = ide.Options{
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

func NewRubyMineServer(userName string, values map[string]config.OptionValue, log log.Logger) *GenericJetBrainsServer {
	amd64Download, arm64Download := getDownloadURLs(RubyMineOptions, values, RubyMineProductCode, RubyMineDownloadAmd64Template, RubyMineDownloadArm64Template)
	return newGenericServer(userName, &GenericOptions{
		ID:            "rubymine",
		DisplayName:   "RubyMine",
		DownloadAmd64: amd64Download,
		DownloadArm64: arm64Download,
	}, log)
}
