package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/inject"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/version"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

var waitForInstanceConnectionTimeout = time.Minute * 5

func InjectAgent(
	ctx context.Context,
	exec inject.ExecFunc,
	local bool,
	remoteAgentPath,
	downloadURL string,
	preferDownload bool,
	log log.Logger,
	timeout time.Duration,
) error {
	return InjectAgentAndExecute(
		ctx,
		exec,
		local,
		remoteAgentPath,
		downloadURL,
		preferDownload,
		"",
		nil,
		nil,
		nil,
		log,
		timeout,
	)
}

func InjectAgentAndExecute(
	ctx context.Context,
	exec inject.ExecFunc,
	local bool,
	remoteAgentPath,
	downloadURL string,
	preferDownload bool,
	command string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	log log.Logger,
	timeout time.Duration,
) error {
	// should execute locally?
	if local {
		if command == "" {
			return nil
		}

		log.Debugf("Execute command locally")
		return shell.ExecuteCommandWithShell(ctx, command, stdin, stdout, stderr, nil)
	}

	defer log.Debugf("Done InjectAgentAndExecute")
	if remoteAgentPath == "" {
		remoteAgentPath = RemoteDevPodHelperLocation
	}
	if downloadURL == "" {
		downloadURL = DefaultAgentDownloadURL()
	}

	versionCheck := fmt.Sprintf(`[ "$(%s version 2>/dev/null || echo 'false')" != "%s" ]`, remoteAgentPath, version.GetVersion())
	if version.GetVersion() == version.DevVersion {
		preferDownload = false
	}

	// install devpod into the target
	// do a simple hello world to check if we can get something
	now := time.Now()
	lastMessage := time.Now()
	for {
		buf := &bytes.Buffer{}
		if stderr != nil {
			stderr = io.MultiWriter(stderr, buf)
		} else {
			stderr = buf
		}

		scriptParams := &inject.Params{
			Command:             command,
			AgentRemotePath:     remoteAgentPath,
			DownloadURLs:        inject.NewDownloadURLs(downloadURL),
			ExistsCheck:         versionCheck,
			PreferAgentDownload: preferDownload,
			ShouldChmodPath:     true,
		}

		wasExecuted, err := inject.InjectAndExecute(
			ctx,
			exec,
			func(arm bool) (io.ReadCloser, error) {
				return injectBinary(arm, downloadURL, log)
			},
			scriptParams,
			stdin,
			stdout,
			stderr,
			timeout,
			log,
		)
		if err != nil {
			if time.Since(now) > waitForInstanceConnectionTimeout {
				return errors.Wrap(err, "timeout waiting for instance connection")
			} else if wasExecuted {
				return errors.Wrapf(err, "agent error: %s", buf.String())
			}

			if time.Since(lastMessage) > time.Second*5 {
				log.Infof("Waiting for devpod agent to come up...")
				lastMessage = time.Now()
			}

			log.Debugf("Inject Error: %s%v", buf.String(), err)
			time.Sleep(time.Second * 3)
			continue
		}

		break
	}

	return nil
}

func injectBinary(arm bool, tryDownloadURL string, log log.Logger) (io.ReadCloser, error) {
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
		if err != nil {
			return nil, errors.Wrap(err, "get executable")
		}

		// check if we still exist
		_, err = os.Stat(binaryPath)
		if err != nil {
			binaryPath = ""
		}
	}

	// download devpod locally
	if binaryPath == "" {
		binaryPath, err = downloadAgentLocally(tryDownloadURL, targetArch, log)
		if err != nil {
			return nil, errors.Wrap(err, "download agent locally")
		}
	}

	// read file
	file, err := os.Open(binaryPath)
	if err != nil {
		return nil, errors.Wrap(err, "open agent binary")
	}

	return file, nil
}

func downloadAgentLocally(tryDownloadURL, targetArch string, log log.Logger) (string, error) {
	agentPath := filepath.Join(os.TempDir(), "devpod-cache", "devpod-linux-"+targetArch)
	err := os.MkdirAll(filepath.Dir(agentPath), 0755)
	if err != nil {
		return "", errors.Wrap(err, "create agent path")
	}

	stat, statErr := os.Stat(agentPath)
	if version.GetVersion() == version.DevVersion && statErr == nil {
		return agentPath, nil
	}

	fullDownloadURL := tryDownloadURL + "/devpod-linux-" + targetArch
	log.Debugf("Attempting to download DevPod agent from: %s", fullDownloadURL)

	resp, err := devpodhttp.GetHTTPClient().Get(fullDownloadURL)
	if err != nil {
		return "", errors.Wrap(err, "download devpod")
	}
	defer resp.Body.Close()

	if statErr == nil && stat.Size() == resp.ContentLength {
		return agentPath, nil
	}

	log.Infof("Download DevPod Agent...")
	file, err := os.Create(agentPath)
	if err != nil {
		return "", errors.Wrap(err, "create agent binary")
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		_ = os.Remove(agentPath)
		return "", errors.Wrapf(err, "failed to download devpod from URL: %s", fullDownloadURL)
	}

	return agentPath, nil
}
