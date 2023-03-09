package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/loft-sh/devpod/pkg/provider"

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

	homeDir, err := homedir.Dir()
	if err != nil {
		return errors.Wrap(err, "get home dir")
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	newFile, err := addHost(sshConfigPath, workspace+"."+"devpod", user, context, workspace, command)
	if err != nil {
		return errors.Wrap(err, "parse ssh config")
	}

	err = os.MkdirAll(filepath.Dir(sshConfigPath), 0755)
	if err != nil {
		log.Debugf("error creating ssh directory: %v", err)
	}

	err = os.WriteFile(sshConfigPath, []byte(newFile), 0600)
	if err != nil {
		return errors.Wrap(err, "write ssh config")
	}

	return nil
}

type DevPodSSHEntry struct {
	Host      string
	User      string
	Workspace string
}

func addHost(path, host, user, context, workspace, command string) (string, error) {
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
	startMarker := "# DevPod Start " + host
	endMarker := "# DevPod End " + host
	for configScanner.Scan() {
		text := configScanner.Text()
		if strings.HasPrefix(text, startMarker) {
			inSection = true
		} else if strings.HasPrefix(text, endMarker) {
			inSection = false
		} else if !inSection {
			newLines = append(newLines, text)
		}
	}
	if configScanner.Err() != nil {
		return "", errors.Wrap(err, "parse ssh config")
	}

	// get path to executable
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// get private key path
	workspaceDir, err := provider.GetWorkspaceDir(context, workspace)
	if err != nil {
		return "", err
	}

	// add new section
	newLines = append(newLines, startMarker)
	newLines = append(newLines, "Host "+host)
	newLines = append(newLines, "  LogLevel error")
	newLines = append(newLines, "  IdentityFile \""+filepath.Join(workspaceDir, DevPodSSHPrivateKeyFile)+"\"")
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
