package jetbrains

import "github.com/loft-sh/devpod/pkg/log"

const RiderDownloadAmd64 = "https://download.jetbrains.com/rider/JetBrains.Rider-2022.3.2.tar.gz"
const RiderDownloadArm64 = "https://download.jetbrains.com/rider/JetBrains.Rider-2022.3.2-aarch64.tar.gz"

func NewRiderServer(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "rider",
		DisplayName:   "Rider",
		DownloadAmd64: RiderDownloadAmd64,
		DownloadArm64: RiderDownloadArm64,
	}, log)
}
