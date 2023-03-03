package jetbrains

import (
	"github.com/loft-sh/devpod/pkg/log"
)

const GolandDownloadAmd64 = "https://download.jetbrains.com/go/goland-2022.3.2.tar.gz"
const GolandDownloadArm64 = "https://download.jetbrains.com/go/goland-2022.3.2-aarch64.tar.gz"

func NewGolandServer(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "goland",
		DisplayName:   "Goland",
		DownloadAmd64: GolandDownloadAmd64,
		DownloadArm64: GolandDownloadArm64,
	}, log)
}
