package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/scanner"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var configLock sync.Mutex

var (
	MarkerStartPrefix = "# DevPod Start "
	MarkerEndPrefix   = "# DevPod End "
)

func ConfigureSSHConfig(context, workspace, user string, log log.Logger) error {
	return configureSSHConfigSameFile(context, workspace, user, "", log)
}

func configureSSHConfigSameFile(context, workspace, user, command string, log log.Logger) error {
	configLock.Lock()
	defer configLock.Unlock()

	sshConfigPath, err := getSSHConfig()
	if err != nil {
		return err
	}

	newFile, err := addHost(sshConfigPath, workspace+"."+"devpod", user, context, workspace, command)
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

func addHost(path, host, user, context, workspace, command string) (string, error) {
	newConfig, err := removeFromConfig(path, host)
	if err != nil {
		return "", err
	}
	newLines := []string{newConfig}

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
	newLines = append(newLines, "  IdentityFile \""+filepath.Join(GetDevPodKeysDir(), DevPodSSHPrivateKeyFile)+"\"")
	newLines = append(newLines, "  StrictHostKeyChecking no")
	newLines = append(newLines, "  UserKnownHostsFile /dev/null")
	newLines = append(newLines, "  IdentitiesOnly yes")
	if command != "" {
		newLines = append(newLines, fmt.Sprintf("  ProxyCommand %s", command))
	} else {
		newLines = append(newLines, fmt.Sprintf("  ProxyCommand %s ssh --stdio --context %s --user %s %s", execPath, context, user, workspace))
	}
	newLines = append(newLines, "  User "+user)
	newLines = append(newLines, endMarker)
	return strings.Join(newLines, "\n"), nil
}

func GetUser(workspace string) (string, error) {
	sshConfigPath, err := getSSHConfig()
	if err != nil {
		return "", err
	}

	user := "root"
	_, err = transformHostSection(sshConfigPath, workspace+"."+"devpod", func(line string) string {
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

func RemoveFromConfig(workspace string, log log.Logger) error {
	configLock.Lock()
	defer configLock.Unlock()

	sshConfigPath, err := getSSHConfig()
	if err != nil {
		return err
	}

	newFile, err := removeFromConfig(sshConfigPath, workspace+"."+"devpod")
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

func getSSHConfig() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "get home dir")
	}

	return filepath.Join(homeDir, ".ssh", "config"), nil
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

	return strings.Join(newLines, "\n"), nil
}
