package gitcredentials

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/pkg/errors"
)

type GitCredentials struct {
	Protocol string `json:"protocol,omitempty"`
	URL      string `json:"url,omitempty"`
	Host     string `json:"host,omitempty"`
	Path     string `json:"path,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type GitUser struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
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

		err = file.Chown(userName, gitConfigPath)
		if err != nil {
			return err
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
	return RemoveHelperFromPath(gitConfigPath)
}

func RemoveHelperFromPath(gitConfigPath string) error {
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
			credentials.URL = strings.Join(splitted[1:], "=")
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
	if credentials.URL != "" {
		request = append(request, "url="+credentials.URL)
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

func SetUser(userName string, user *GitUser) error {
	if user.Name != "" {
		command := fmt.Sprintf("git config --global user.name '%s'", user.Name)
		args := []string{}
		if userName != "" {
			args = append(args, "su", userName, "-c", command)
		} else {
			args = append(args, "sh", "-c", command)
		}

		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return errors.Wrapf(err, "set user.name: %s", string(out))
		}
	}
	if user.Email != "" {
		command := fmt.Sprintf("git config --global user.email '%s'", user.Email)
		args := []string{}
		if userName != "" {
			args = append(args, "su", userName, "-c", command)
		} else {
			args = append(args, "sh", "-c", command)
		}

		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return errors.Wrapf(err, "set user.email: %s", string(out))
		}
	}
	return nil
}

func GetUser() (*GitUser, error) {
	gitUser := &GitUser{}

	// we ignore the error here, because if email is empty we don't care
	name, _ := exec.Command("git", "config", "--global", "user.name").Output()
	gitUser.Name = strings.TrimSpace(string(name))

	email, _ := exec.Command("git", "config", "--global", "user.email").Output()
	gitUser.Email = strings.TrimSpace(string(email))
	return gitUser, nil
}

func GetCredentials(requestObj *GitCredentials) (*GitCredentials, error) {
	var c *exec.Cmd

	gitHelperPort := os.Getenv("DEVPOD_GIT_HELPER_PORT")
	if gitHelperPort != "" {
		binaryPath, err := os.Executable()
		if err != nil {
			return nil, err
		}

		c = exec.Command(binaryPath, "agent", "git-credentials", "--port", gitHelperPort, "get")
	} else {
		c = git.CommandContext(context.TODO(), "credential", "fill")
	}

	c.Stdin = strings.NewReader(ToString(requestObj))
	stdout, err := c.Output()
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
