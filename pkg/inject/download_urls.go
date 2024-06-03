package inject

import (
	"net/url"
	"strings"
)

const AmdUrl = "devpod-linux-amd64"
const ArmUrl = "devpod-linux-arm64"
const BinNamePlaceholder = "${BIN_NAME}"

type DownloadURLs struct {
	Base string
	Amd  string
	Arm  string
}

func NewDownloadURLs(baseUrl string) *DownloadURLs {
	var amdUrl, armUrl string

	// replace ${BIN_NAME} with binary name
	if strings.Contains(baseUrl, BinNamePlaceholder) {
		baseUrl = strings.TrimSuffix(baseUrl, "/")
		amdUrl = strings.Replace(baseUrl, BinNamePlaceholder, AmdUrl, 1)
		armUrl = strings.Replace(baseUrl, BinNamePlaceholder, ArmUrl, 1)
	} else {
		amdUrl, _ = url.JoinPath(baseUrl, AmdUrl)
		armUrl, _ = url.JoinPath(baseUrl, ArmUrl)
	}

	return &DownloadURLs{
		Base: baseUrl,
		Amd:  amdUrl,
		Arm:  armUrl,
	}
}
