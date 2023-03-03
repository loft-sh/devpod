package jetbrains

import "github.com/loft-sh/devpod/pkg/log"

const WebStormDownloadAmd64 = "https://download.jetbrains.com/webstorm/WebStorm-2022.3.2.tar.gz"
const WebStormDownloadArm64 = "https://download.jetbrains.com/webstorm/WebStorm-2022.3.2-aarch64.tar.gz"

func NewWebStormServer(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "webstorm",
		DisplayName:   "WebStorm",
		DownloadAmd64: WebStormDownloadAmd64,
		DownloadArm64: WebStormDownloadArm64,
	}, log)
}
