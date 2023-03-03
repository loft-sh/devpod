package jetbrains

import "github.com/loft-sh/devpod/pkg/log"

const IntellijDownloadAmd64 = "https://download.jetbrains.com/idea/ideaIU-2022.3.2.tar.gz"
const IntellijDownloadArm64 = "https://download.jetbrains.com/idea/ideaIU-2022.3.2-aarch64.tar.gz"

func NewIntellij(userName string, log log.Logger) *GenericJetBrainsServer {
	return newGenericServer(userName, &GenericOptions{
		ID:            "intellij",
		DisplayName:   "Intellij",
		DownloadAmd64: IntellijDownloadAmd64,
		DownloadArm64: IntellijDownloadArm64,
	}, log)
}
