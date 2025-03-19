package gitcredentials

import (
	"context"
	"fmt"
	netUrl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/log/scanner"
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
	gitConfigPath, err := getGlobalGitConfigPath(userName)
	if err != nil {
		return err
	}

	out, err := os.ReadFile(gitConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	helper := fmt.Sprintf(`helper = !'%s' agent git-credentials`, binaryPath)
	if port != -1 {
		helper += fmt.Sprintf(` --port %d`, port)
	}

	config := string(out)
	if !strings.Contains(config, helper) {
		content := removeCredentialHelper(config) + fmt.Sprintf(`
[credential]
        %s
`, helper)

		err = os.WriteFile(gitConfigPath, []byte(content), 0600)
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
	gitConfigPath, err := getGlobalGitConfigPath(userName)
	if err != nil {
		return err
	}

	return RemoveHelperFromPath(gitConfigPath)
}

func RemoveHelperFromPath(gitConfigPath string) error {
	out, err := os.ReadFile(gitConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if strings.Contains(string(out), "[credential]") {
		err = os.WriteFile(gitConfigPath, []byte(removeCredentialHelper(string(out))), 0600)
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
		shellCommand := fmt.Sprintf(`git config --global user.name "%s"`, user.Name)
		args := []string{}
		if userName != "" {
			args = append(args, "su", userName, "-c", shellCommand)
		} else {
			args = append(args, "sh", "-c", shellCommand)
		}

		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("set user.name '%s': %w", strings.Join(args, " "), command.WrapCommandError(out, err))
		}
	}
	if user.Email != "" {
		shellCommand := fmt.Sprintf(`git config --global user.email "%s"`, user.Email)
		args := []string{}
		if userName != "" {
			args = append(args, "su", userName, "-c", shellCommand)
		} else {
			args = append(args, "sh", "-c", shellCommand)
		}

		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("set user.email '%s': %w", strings.Join(args, " "), command.WrapCommandError(out, err))
		}
	}
	return nil
}

func GetUser(userName string) (*GitUser, error) {
	gitUser := &GitUser{}

	scopeArgs := []string{"config"}
	if userName != "" {
		p, err := getGlobalGitConfigPath(userName)
		if err != nil {
			return gitUser, fmt.Errorf("get git global dir for %s: %w", userName, err)
		}
		scopeArgs = append(scopeArgs, "--file", p)
	} else {
		scopeArgs = append(scopeArgs, "--global")
	}

	// we ignore the error here, because if email is empty we don't care
	name, _ := exec.Command("git", append(scopeArgs[:], "user.name")...).Output()
	gitUser.Name = strings.TrimSpace(string(name))

	email, _ := exec.Command("git", append(scopeArgs[:], "user.email")...).Output()
	gitUser.Email = strings.TrimSpace(string(email))

	return gitUser, nil
}

func GetCredentials(requestObj *GitCredentials) (*GitCredentials, error) {
	// run in git helper mode if we have a port
	gitHelperPort := os.Getenv("DEVPOD_GIT_HELPER_PORT")
	if gitHelperPort != "" {
		binaryPath, err := os.Executable()
		if err != nil {
			return nil, err
		}

		c := exec.Command(binaryPath, "agent", "git-credentials", "--port", gitHelperPort, "get")

		c.Stdin = strings.NewReader(ToString(requestObj))
		stdout, err := c.Output()
		if err != nil {
			return nil, err
		}

		return Parse(string(stdout))
	}

	// use local credentials if not
	c := git.CommandContext(context.TODO(), git.GetDefaultExtraEnv(false), "credential", "fill")
	c.Stdin = strings.NewReader(ToString(requestObj))
	stdout, err := c.Output()
	if err != nil {
		return nil, err
	}
	return Parse(string(stdout))
}

type GetHttpPathParameters struct {
	Host        string
	Protocol    string
	CurrentPath string
	Repository  string
}

// GetHTTPPath checks for gits `credential.useHttpPath` setting for a given host+protocol and returns the path component
// of `GitCredential` if the setting is true
func GetHTTPPath(ctx context.Context, params GetHttpPathParameters) (string, error) {
	// No need to look up the HTTP Path if we already have one
	if params.CurrentPath != "" {
		return params.CurrentPath, nil
	}

	// Check if we need to respect gits `credential.useHttpPath`
	// The actual format for the key is `credential.$PROTOCOL://$HOST.useHttpPath`, i.e. `credential.https://github.com.useHttpPath`
	// We need to ignore the error as git will always exit with 1 if the key doesn't exist
	configKey := fmt.Sprintf("credential.%s://%s.useHttpPath", params.Protocol, params.Host)
	out, _ := git.CommandContext(ctx, git.GetDefaultExtraEnv(false), "config", "--get", configKey).Output()
	if strings.TrimSpace(string(out)) != "true" {
		return "", nil
	}
	// We can assume the GitRepository is always HTTP(S) based as otherwise we wouldn't
	// request credentials for it
	url, err := netUrl.Parse(params.Repository)
	if err != nil {
		return "", fmt.Errorf("parse workspace repository: %w", err)
	}

	return url.Path, nil
}

func SetupGpgGitKey(gitSignKey string) error {
	gitConfigCmd := exec.Command("git", []string{"config", "--global", "user.signingKey", gitSignKey}...)

	out, err := gitConfigCmd.Output()
	if err != nil {
		return errors.Wrap(err, "git signkey: "+string(out))
	}

	return nil
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

// getGlobalGitConfigPath resolves the global git config for the specified user according to
// https://git-scm.com/docs/git-config/#Documentation/git-config.txt-XDGCONFIGHOMEgitconfig
func getGlobalGitConfigPath(userName string) (string, error) {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "git", "config"), nil
	}

	home, err := command.GetHome(userName)
	if err != nil {
		return "", fmt.Errorf("get homedir for %s: %w", userName, err)
	}

	return filepath.Join(home, ".gitconfig"), nil
}

// GetLocalGitConfigPath resolves the local git config for the specified repository path
func GetLocalGitConfigPath(repoPath string) string {
	return filepath.Join(repoPath, ".git", "config")
}
