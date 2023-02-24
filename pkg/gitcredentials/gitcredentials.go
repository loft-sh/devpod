package gitcredentials

import (
	"context"
	"fmt"
	"github.com/gofrs/flock"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type GitCredentials struct {
	Protocol string `json:"protocol,omitempty"`
	Url      string `json:"url,omitempty"`
	Host     string `json:"host,omitempty"`
	Path     string `json:"path,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func ConfigureHelper(binaryPath, userName string, port int) error {
	homeDir, err := command.GetHome(userName)
	if err != nil {
		return err
	}

	gitConfigPath := filepath.Join(homeDir, ".gitconfig")
	out, err := os.ReadFile(gitConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	config := string(out)
	if !strings.Contains(config, fmt.Sprintf(`helper = "%s agent git-credentials --port %d"`, binaryPath, port)) {
		content := removeCredentialHelper(config) + fmt.Sprintf(`
[credential]
        helper = "%s agent git-credentials --port %d"
`, binaryPath, port)

		err = os.WriteFile(gitConfigPath, []byte(content), 0644)
		if err != nil {
			return errors.Wrap(err, "write git config")
		}
	}

	return nil
}

func RemoveHelper(userName string) error {
	homeDir, err := command.GetHome(userName)
	if err != nil {
		return err
	}

	gitConfigPath := filepath.Join(homeDir, ".gitconfig")
	out, err := os.ReadFile(gitConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if strings.Contains(string(out), "[credential]") {
		err = os.WriteFile(gitConfigPath, []byte(removeCredentialHelper(string(out))), 0644)
		if err != nil {
			return errors.Wrap(err, "write git config")
		}
	}

	return nil
}

func Parse(raw string) (*GitCredentials, error) {
	credentials := &GitCredentials{}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		splitted := strings.Split(line, "=")
		if len(splitted) == 1 {
			continue
		}

		if splitted[0] == "protocol" {
			credentials.Protocol = strings.Join(splitted[1:], "=")
		} else if splitted[0] == "host" {
			credentials.Host = strings.Join(splitted[1:], "=")
		} else if splitted[0] == "username" {
			credentials.Username = strings.Join(splitted[1:], "=")
		} else if splitted[0] == "password" {
			credentials.Password = strings.Join(splitted[1:], "=")
		} else if splitted[0] == "url" {
			credentials.Url = strings.Join(splitted[1:], "=")
		} else if splitted[0] == "path" {
			credentials.Path = strings.Join(splitted[1:], "=")
		}
	}

	return credentials, nil
}

func ToString(credentials *GitCredentials) string {
	request := []string{}
	if credentials.Protocol != "" {
		request = append(request, "protocol="+credentials.Protocol)
	}
	if credentials.Url != "" {
		request = append(request, "url="+credentials.Url)
	}
	if credentials.Path != "" {
		request = append(request, "path="+credentials.Path)
	}
	if credentials.Host != "" {
		request = append(request, "host="+credentials.Host)
	}
	if credentials.Username != "" {
		request = append(request, "username="+credentials.Username)
	}
	if credentials.Password != "" {
		request = append(request, "password="+credentials.Password)
	}

	return strings.Join(request, "\n") + "\n"
}

func GetCredentials(requestObj *GitCredentials) (*GitCredentials, error) {
	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader(ToString(requestObj))
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return Parse(string(stdout))
}

func removeCredentialHelper(content string) string {
	scan := scanner.NewScanner(strings.NewReader(content))

	isCredential := false
	out := []string{}
	for scan.Scan() {
		line := scan.Text()
		if strings.TrimSpace(line) == "[credential]" {
			isCredential = true
			continue
		} else if isCredential {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 && trimmed[0] == '[' {
				isCredential = false
			} else {
				continue
			}
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func RunCredentialsServer(ctx context.Context, userName string, port int, client tunnel.TunnelClient, log log.Logger) error {
	fileLock := flock.New(filepath.Join(os.TempDir(), "devpod-credentials.lock"))
	locked, err := fileLock.TryLock()
	if err != nil {
		return errors.Wrap(err, "acquire lock")
	} else if !locked {
		return nil
	}
	defer fileLock.Unlock()

	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	err = ConfigureHelper(binaryPath, userName, port)
	if err != nil {
		return errors.Wrap(err, "configure git helper")
	}

	// cleanup when we are done
	defer func() {
		_ = RemoveHelper(userName)
	}()

	srv := &http.Server{
		Addr: "localhost:" + strconv.Itoa(port),
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			log.Infof("Incoming client connection at %s", request.URL.Path)
			if request.URL.Path != "/git-credentials" {
				return
			}

			err := handleCredentialsRequest(ctx, writer, request, client, log)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		}),
	}

	errChan := make(chan error, 1)
	go func() {
		log.Infof("Credentials server started on %d...", port)

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		} else {
			errChan <- nil
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		_ = srv.Close()
		return nil
	}
}

func handleCredentialsRequest(ctx context.Context, writer http.ResponseWriter, request *http.Request, client tunnel.TunnelClient, log log.Logger) error {
	out, err := io.ReadAll(request.Body)
	if err != nil {
		return errors.Wrap(err, "read request body")
	}

	log.Debugf("Received credentials post data: %s", string(out))
	response, err := client.GitCredentials(ctx, &tunnel.Message{Message: string(out)})
	if err != nil {
		return errors.Wrap(err, "get git credentials response")
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(response.Message))
	log.Debugf("Successfully wrote back %d bytes", len(response.Message))
	return nil
}
