package inject

import (
	"net/url"
)

const AmdUrl = "devpod-linux-amd64"
const ArmUrl = "devpod-linux-arm64"

type DownloadURLs struct {
	Base string
	Amd  string
	Arm  string
}

func NewDownloadURLs(baseUrl string) *DownloadURLs {
	amdUrl, _ := url.JoinPath(baseUrl, AmdUrl)
	armUrl, _ := url.JoinPath(baseUrl, ArmUrl)

	return &DownloadURLs{
		Base: baseUrl,
		Amd:  amdUrl,
		Arm:  armUrl,
	}
}
