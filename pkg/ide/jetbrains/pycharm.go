package jetbrains

import "github.com/loft-sh/devpod/pkg/log"

const PycharmDownloadAmd64 = "https://download.jetbrains.com/python/pycharm-professional-2022.3.2.tar.gz"
const PycharmDownloadArm64 = "https://download.jetbrains.com/python/pycharm-professional-2022.3.2-aarch64.tar.gz"

func NewPyCharmServer(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "pycharm",
		DisplayName:   "PyCharm",
		DownloadAmd64: PycharmDownloadAmd64,
		DownloadArm64: PycharmDownloadArm64,
	}, log)
}
