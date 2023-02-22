package agent

import (
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/inject"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func InjectAgent(exec inject.ExecFunc, remoteAgentPath, downloadURL string, preferDownload bool, timeout time.Duration) error {
	if remoteAgentPath == "" {
		remoteAgentPath = RemoteDevPodHelperLocation
	}
	if downloadURL == "" {
		downloadURL = DefaultAgentDownloadURL
	}

	err := inject.Inject(
		exec,
		func(arm bool) (io.ReadCloser, error) {
			return injectBinary(arm, downloadURL)
		},
		fmt.Sprintf(`[ "$(%s version && echo 'true' || echo 'false')" = "false" ]`, remoteAgentPath),
		remoteAgentPath,
		downloadURL+"/devpod-linux-amd64",
		downloadURL+"/devpod-linux-arm64",
		preferDownload,
		true,
		timeout,
	)
	return err
}

func injectBinary(arm bool, tryDownloadURL string) (io.ReadCloser, error) {
	// this means we need to
	targetArch := "amd64"
	if arm {
		targetArch = "arm64"
	}

	// make sure a linux arm64 binary exists locally
	var err error
	var binaryPath string
	if runtime.GOOS == "linux" && runtime.GOARCH == targetArch {
		binaryPath, err = os.Executable()
	} else {
		binaryPath, err = downloadAgentLocally(tryDownloadURL, targetArch)
	}
	if err != nil {
		return nil, err
	}

	// read file
	file, err := os.Open(binaryPath)
	if err != nil {
		return nil, errors.Wrap(err, "open agent binary")
	}

	return file, nil
}

func downloadAgentLocally(tryDownloadURL, targetArch string) (string, error) {
	agentPath := filepath.Join(os.TempDir(), "devpod-cache", "devpod-linux-"+targetArch)
	_, err := os.Stat(agentPath)
	if err == nil {
		return agentPath, nil
	}

	err = os.MkdirAll(filepath.Dir(agentPath), 0755)
	if err != nil {
		return "", errors.Wrap(err, "create agent path")
	}

	file, err := os.Create(agentPath)
	if err != nil {
		return "", errors.Wrap(err, "create agent binary")
	}
	defer file.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := httpClient.Get(tryDownloadURL + "/devpod-linux-" + targetArch)
	if err != nil {
		return "", errors.Wrap(err, "download devpod")
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		_ = os.Remove(agentPath)
		return "", errors.Wrap(err, "download devpod")
	}

	return agentPath, nil
}
