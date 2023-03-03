package jetbrains

import "github.com/loft-sh/devpod/pkg/log"

const RubyMineDownloadAmd64 = "https://download.jetbrains.com/ruby/RubyMine-2022.3.2.tar.gz"
const RubyMineDownloadArm64 = "https://download.jetbrains.com/ruby/RubyMine-2022.3.2-aarch64.tar.gz"

func NewRubyMineServer(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "rubymine",
		DisplayName:   "RubyMine",
		DownloadAmd64: RubyMineDownloadAmd64,
		DownloadArm64: RubyMineDownloadArm64,
	}, log)
}
