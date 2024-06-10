package loft

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/provider"
)

type VersionObject struct {
	// Version is the remote devpod version
	DevPodVersion string `json:"devPodVersion,omitempty"`
}

func GetProInstanceDevPodVersion(proInstance *provider.ProInstance) (string, error) {
	url := "https://" + proInstance.Host
	return GetDevPodVersion(url)
}

func GetDevPodVersion(url string) (string, error) {
	resp, err := http.GetHTTPClient().Get(url + "/version")
	if err != nil {
		return "", fmt.Errorf("get %s: %w", url, err)
	} else if resp.StatusCode != 200 {
		out, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get %s: %s (Status: %d)", url, string(out), resp.StatusCode)
	}

	versionRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}

	version := &VersionObject{}
	err = json.Unmarshal(versionRaw, version)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", url, err)
	} else if version.DevPodVersion == "" {
		return "", fmt.Errorf("unexpected version '%s', please use --version to define a provider version", version.DevPodVersion)
	}

	// make sure it starts with a v
	if !strings.HasPrefix(version.DevPodVersion, "v") {
		version.DevPodVersion = "v" + version.DevPodVersion
	}

	return version.DevPodVersion, nil
}
