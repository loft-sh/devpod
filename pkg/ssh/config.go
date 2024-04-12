package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/scanner"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var configLock sync.Mutex

var (
	MarkerStartPrefix = "# DevPod Start "
	MarkerEndPrefix   = "# DevPod End "
)

func ConfigureSSHConfig(sshConfigPath, context, workspace, user, workdir string, gpgagent bool, log log.Logger) error {
	return configureSSHConfigSameFile(sshConfigPath, context, workspace, user, workdir, "", gpgagent, log)
}

func configureSSHConfigSameFile(sshConfigPath, context, workspace, user, workdir, command string, gpgagent bool, log log.Logger) error {
	configLock.Lock()
	defer configLock.Unlock()

	newFile, err := addHost(sshConfigPath, workspace+"."+"devpod", user, context, workspace, workdir, command, gpgagent)
	if err != nil {
		return errors.Wrap(err, "parse ssh config")
	}

	return writeSSHConfig(sshConfigPath, newFile, log)
}

type DevPodSSHEntry struct {
	Host      string
	User      string
	Workspace string
}

func addHost(path, host, user, context, workspace, workdir, command string, gpgagent bool) (string, error) {
	newConfig, err := removeFromConfig(path, host)
	if err != nil {
		return "", err
	}
	newLines := []string{}

	// get path to executable
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// add new section
	startMarker := MarkerStartPrefix + host
	endMarker := MarkerEndPrefix + host
	newLines = append(newLines, startMarker)
	newLines = append(newLines, "Host "+host)
	newLines = append(newLines, "  ForwardAgent yes")
	newLines = append(newLines, "  LogLevel error")
	newLines = append(newLines, "  StrictHostKeyChecking no")
	newLines = append(newLines, "  UserKnownHostsFile /dev/null")
	newLines = append(newLines, "  HostKeyAlgorithms rsa-sha2-256,rsa-sha2-512,ssh-rsa")
	if command != "" {
		newLines = append(newLines, fmt.Sprintf("  ProxyCommand \"%s\"", command))
	} else if gpgagent {
		newLines = append(newLines, fmt.Sprintf("  ProxyCommand \"%s\" ssh --gpg-agent-forwarding --stdio --context %s --user %s %s", execPath, context, user, workspace))
	} else {
		proxyCommand := fmt.Sprintf("  ProxyCommand \"%s\" ssh --stdio --context %s --user %s %s", execPath, context, user, workspace)
		if workdir != "" {
			proxyCommand = fmt.Sprintf("%s --workdir %s", proxyCommand, workdir)
		}
		newLines = append(newLines, proxyCommand)
	}
	newLines = append(newLines, "  User "+user)
	newLines = append(newLines, endMarker)
	// add a space between blocks
	newLines = append(newLines, "")

	// now we append the original config
	// keep our blocks on top of the file for priority reasons
	newLines = append(newLines, newConfig)
	return strings.Join(newLines, "\n"), nil
}

func GetUser(workspaceID string, sshConfigPath string) (string, error) {
	path, err := ResolveSSHConfigPath(sshConfigPath)
	if err != nil {
		return "", errors.Wrap(err, "Invalid ssh config path")
	}
	sshConfigPath = path

	user := "root"
	_, err = transformHostSection(sshConfigPath, workspaceID+"."+"devpod", func(line string) string {
		splitted := strings.Split(strings.ToLower(strings.TrimSpace(line)), " ")
		if len(splitted) == 2 && splitted[0] == "user" {
			user = strings.Trim(splitted[1], "\"")
		}

		return line
	})
	if err != nil {
		return "", err
	}

	return user, nil
}

func RemoveFromConfig(workspaceID string, sshConfigPath string, log log.Logger) error {
	configLock.Lock()
	defer configLock.Unlock()

	newFile, err := removeFromConfig(sshConfigPath, workspaceID+"."+"devpod")
	if err != nil {
		return errors.Wrap(err, "parse ssh config")
	}

	return writeSSHConfig(sshConfigPath, newFile, log)
}

func writeSSHConfig(path, content string, log log.Logger) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		log.Debugf("error creating ssh directory: %v", err)
	}

	err = os.WriteFile(path, []byte(content), 0600)
	if err != nil {
		return errors.Wrap(err, "write ssh config")
	}

	return nil
}

func ResolveSSHConfigPath(sshConfigPath string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "get home dir")
	}

	if sshConfigPath == "" {
		return filepath.Join(homeDir, ".ssh", "config"), nil
	}

	if strings.HasPrefix(sshConfigPath, "~/") {
		sshConfigPath = strings.Replace(sshConfigPath, "~", homeDir, 1)
	}

	return filepath.Abs(sshConfigPath)
}

func removeFromConfig(path, host string) (string, error) {
	return transformHostSection(path, host, func(line string) string {
		return ""
	})
}

func transformHostSection(path, host string, transform func(line string) string) (string, error) {
	var reader io.Reader
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		reader = strings.NewReader("")
	} else {
		reader = f
		defer f.Close()
	}

	configScanner := scanner.NewScanner(reader)
	newLines := []string{}
	inSection := false
	startMarker := MarkerStartPrefix + host
	endMarker := MarkerEndPrefix + host
	for configScanner.Scan() {
		text := configScanner.Text()
		if strings.HasPrefix(text, startMarker) {
			inSection = true
		} else if strings.HasPrefix(text, endMarker) {
			inSection = false
		} else if !inSection {
			newLines = append(newLines, text)
		} else if inSection {
			text = transform(text)
			if text != "" {
				newLines = append(newLines, text)
			}
		}
	}
	if configScanner.Err() != nil {
		return "", errors.Wrap(err, "parse ssh config")
	}

	// remove residual empty line at start file
	if len(newLines) > 0 && newLines[0] == "" {
		newLines = newLines[1:]
	}

	return strings.Join(newLines, "\n"), nil
}
