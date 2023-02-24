package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/inject"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var waitForInstanceConnectionTimeout = time.Minute * 5

func InjectAgent(ctx context.Context, exec inject.ExecFunc, remoteAgentPath, downloadURL string, preferDownload bool, log log.Logger) error {
	return InjectAgentAndExecute(ctx, exec, remoteAgentPath, downloadURL, preferDownload, "", nil, nil, nil, log)
}

func InjectAgentAndExecute(ctx context.Context, exec inject.ExecFunc, remoteAgentPath, downloadURL string, preferDownload bool, command string, stdin io.Reader, stdout io.Writer, stderr io.Writer, log log.Logger) error {
	if remoteAgentPath == "" {
		remoteAgentPath = RemoteDevPodHelperLocation
	}
	if downloadURL == "" {
		downloadURL = DefaultAgentDownloadURL
	}

	// install devpod into the target
	// do a simple hello world to check if we can get something
	startWaiting := time.Now()
	now := startWaiting
	for {
		err := inject.InjectAndExecute(
			ctx,
			exec,
			func(arm bool) (io.ReadCloser, error) {
				return injectBinary(arm, downloadURL)
			},
			fmt.Sprintf(`[ "$(%s version >/dev/null 2>&1 && echo 'true' || echo 'false')" = "false" ]`, remoteAgentPath),
			remoteAgentPath,
			downloadURL+"/devpod-linux-amd64",
			downloadURL+"/devpod-linux-arm64",
			preferDownload,
			true,
			command,
			stdin,
			stdout,
			stderr,
			time.Second*10,
			log,
		)
		if err != nil {
			if time.Since(now) > waitForInstanceConnectionTimeout {
				return errors.Wrap(err, "timeout waiting for instance connection")
			} else if strings.HasPrefix(err.Error(), "unexpected start line: ") || err == context.DeadlineExceeded {
				log.Infof("Waiting for devpod agent to come up...")
				log.Debugf("Inject Error: %v", err)
				startWaiting = time.Now()
				continue
			}

			return err
		}

		break
	}

	return nil
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
