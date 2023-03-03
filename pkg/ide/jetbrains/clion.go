package jetbrains

import "github.com/loft-sh/devpod/pkg/log"

const CLionDownloadAmd64 = "https://download.jetbrains.com/cpp/CLion-2022.3.2.tar.gz"
const CLionDownloadArm64 = "https://download.jetbrains.com/cpp/CLion-2022.3.2-aarch64.tar.gz"

func NewCLionServer(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "clion",
		DisplayName:   "CLion",
		DownloadAmd64: CLionDownloadAmd64,
		DownloadArm64: CLionDownloadArm64,
	}, log)
}
