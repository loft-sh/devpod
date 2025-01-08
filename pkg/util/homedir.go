package util

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// UserHomeDir returns the home directory for the executing user.
//
// This extends the logic of os.UserHomeDir() with the now archived package
// github.com/mitchellh/go-homedir for compatibility.
func UserHomeDir() (string, error) {
	// Always try the HOME environment variable first
	homeEnv := "HOME"
	if runtime.GOOS == "plan9" {
		homeEnv = "home"
	}
	if home := os.Getenv(homeEnv); home != "" {
		return home, nil
	}

	// Rely on os.UserHomeDir() here, as it's the standard method moving forward
	if home, _ := os.UserHomeDir(); home != "" {
		return home, nil
	}

	var stdout bytes.Buffer

	// Finally, handle cases existed in go-homedir but not in the current
	// os.UserHomeDir() implementation
	switch runtime.GOOS {
	case "windows":
		drive := os.Getenv("HOMEDRIVE")
		path := os.Getenv("HOMEPATH")
		if drive == "" || path == "" {
			return "", errors.New("HOMEDRIVE, HOMEPATH, or USERPROFILE are blank")
		}
		return drive + path, nil
	case "darwin":
		cmd := exec.Command("sh", "-c", `dscl -q . -read /Users/"$(whoami)" NFSHomeDirectory | sed 's/^[^ ]*: //'`)
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			result := strings.TrimSpace(stdout.String())
			if result != "" {
				return result, nil
			}
		}
	default:
		cmd := exec.Command("getent", "passwd", strconv.Itoa(os.Getuid()))
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			// If the error is ErrNotFound, we ignore it. Otherwise, return it.
			if errors.Is(err, exec.ErrNotFound) {
				return "", err
			}
		} else {
			if passwd := strings.TrimSpace(stdout.String()); passwd != "" {
				// username:password:uid:gid:gecos:home:shell
				passwdParts := strings.SplitN(passwd, ":", 7)
				if len(passwdParts) > 5 {
					return passwdParts[5], nil
				}
			}
		}
	}

	// If all else fails, try the shell
	if runtime.GOOS != "windows" {
		stdout.Reset()
		cmd := exec.Command("sh", "-c", "cd && pwd")
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return "", err
		}

		result := strings.TrimSpace(stdout.String())
		if result == "" {
			return "", errors.New("blank output when reading home directory")
		}

		return result, nil
	}

	return "", errors.New("can't determine the home directory")
}
