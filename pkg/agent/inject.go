package agent

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type ExecFunc func(command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error

func InjectAgent(exec ExecFunc, remoteAgentPath, downloadURL string, preferDownload bool) error {
	if remoteAgentPath == "" {
		remoteAgentPath = RemoteDevPodHelperLocation
	}
	if downloadURL == "" {
		downloadURL = DefaultAgentDownloadURL
	}

	// two methods:
	// - Use tar directly if we want to copy current binary
	// - Call small helper script to download binary
	if !preferDownload {
		err := injectBinary(remoteAgentPath, downloadURL, exec)
		if err != nil {
			err := downloadBinaryRemotely(remoteAgentPath, downloadURL, exec)
			if err != nil {
				return fmt.Errorf("error downloading devpod agent into target: %v", err)
			}
		}
	} else {
		err := downloadBinaryRemotely(remoteAgentPath, downloadURL, exec)
		if err != nil {
			err := injectBinary(remoteAgentPath, downloadURL, exec)
			if err != nil {
				return fmt.Errorf("error injecting devpod agent into target: %v", err)
			}
		}
	}

	return nil
}

func downloadBinaryRemotely(remoteAgentPath, tryDownloadURL string, exec ExecFunc) error {
	// use download in this case
	t, err := template.FillTemplate(scripts.InstallDevPodTemplate, map[string]string{
		"BaseUrl":   tryDownloadURL,
		"AgentPath": remoteAgentPath,
	})
	if err != nil {
		return err
	}

	// execute script
	buf := &bytes.Buffer{}
	err = exec(t, nil, buf, buf)
	if err != nil {
		return errors.Wrapf(err, "download agent binary: %s", buf.String())
	}

	return nil
}

func injectBinary(remoteAgentPath, tryDownloadURL string, exec ExecFunc) (err error) {
	// first parse remote
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdinWriter.Close()
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer stdoutWriter.Close()

	// start command
	stderr := &bytes.Buffer{}
	errChan := make(chan error)
	go func() {
		errChan <- exec(fmt.Sprintf(`INSTALLED=$(%s version 2> /dev/null && echo -n "exists" || true)
if [ -z "$INSTALLED" ]; then
  uname -a
  cat > %s && chmod +x %s
fi

echo "done"`, remoteAgentPath, remoteAgentPath, remoteAgentPath), stdinReader, stdoutWriter, stderr)
	}()

	// wait until we read something
	reader := bufio.NewReader(stdoutReader)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return err
	} else if strings.TrimSpace(string(line)) == "done" {
		return nil
	}

	// this means we need to
	targetArch := "amd64"
	if strings.Contains(string(line), "arm64") || strings.Contains(string(line), "aarch64") {
		targetArch = "arm64"
	}

	// make sure a linux arm64 binary exists locally
	var binaryPath string
	if runtime.GOOS == "linux" && runtime.GOARCH == targetArch {
		binaryPath, err = os.Executable()
	} else {
		binaryPath, err = downloadAgentLocally(tryDownloadURL, targetArch)
	}
	if err != nil {
		return err
	}

	// read file
	file, err := os.Open(binaryPath)
	if err != nil {
		return errors.Wrap(err, "open agent binary")
	}
	defer file.Close()

	// pipe into stdin
	_, err = io.Copy(stdinWriter, file)
	if err != nil {
		return errors.Wrap(err, "copy to remote")
	}

	return nil
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
