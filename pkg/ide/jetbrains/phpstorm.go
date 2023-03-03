package jetbrains

import "github.com/loft-sh/devpod/pkg/log"

const PhpStormDownloadAmd64 = "https://download.jetbrains.com/webide/PhpStorm-2022.3.2.tar.gz"
const PhpStormDownloadArm64 = "https://download.jetbrains.com/webide/PhpStorm-2022.3.2-aarch64.tar.gz"

func NewPhpStorm(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "phpstorm",
		DisplayName:   "PhpStorm",
		DownloadAmd64: PhpStormDownloadAmd64,
		DownloadArm64: PhpStormDownloadArm64,
	}, log)
}
