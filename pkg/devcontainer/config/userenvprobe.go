package config

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/log"
)

type UserEnvProbe string

const (
	LoginInteractiveShellProbe UserEnvProbe = "loginInteractiveShell"
	LoginShellProbe            UserEnvProbe = "loginShell"
	InteractiveShellProbe      UserEnvProbe = "interactiveShell"
	NoneProbe                  UserEnvProbe = "none"

	DefaultUserEnvProbe UserEnvProbe = LoginInteractiveShellProbe
)

func NewUserEnvProbe(probe string) (UserEnvProbe, error) {
	switch probe {
	case string(LoginInteractiveShellProbe):
		return LoginInteractiveShellProbe, nil
	case string(LoginShellProbe):
		return LoginShellProbe, nil
	case string(InteractiveShellProbe):
		return InteractiveShellProbe, nil
	case string(NoneProbe):
		return NoneProbe, nil
	case "":
		return DefaultUserEnvProbe, nil
	default:
		return "", fmt.Errorf("invalid userEnvProbe \"%s\", supported are \"%s\"", probe,
			strings.Join([]string{
				string(LoginInteractiveShellProbe),
				string(LoginShellProbe),
				string(InteractiveShellProbe),
				string(NoneProbe),
			}, ","))
	}
}

func ProbeUserEnv(ctx context.Context, probe string, userName string, log log.Logger) (map[string]string, error) {
	userEnvProbe, err := NewUserEnvProbe(probe)
	if err != nil {
		log.Warnf("Get user env probe: %v", err)
		log.Warnf("Falling back to default user env probe: %s", DefaultUserEnvProbe)
		userEnvProbe = DefaultUserEnvProbe
	}
	if userEnvProbe == NoneProbe {
		return map[string]string{}, nil
	}

	preferredShell, err := shell.GetShell(userName)
	if err != nil {
		return nil, fmt.Errorf("find shell for user %s: %w", userName, err)
	}

	log.Debugf("running user env probe with shell \"%s\", probe \"%s\", user \"%s\" and command \"%s\"",
		strings.Join(preferredShell, " "), string(userEnvProbe), userName, "cat /proc/self/environ")

	probedEnv, err := doProbe(ctx, userEnvProbe, preferredShell, userName, "cat /proc/self/environ", '\x00', log)
	if err != nil {
		log.Debugf("running user env probe with shell \"%s\", probe \"%s\", user \"%s\" and command \"%s\"",
			strings.Join(preferredShell, " "), string(userEnvProbe), userName, "printenv")

		newProbedEnv, newErr := doProbe(ctx, userEnvProbe, preferredShell, userName, "printenv", '\n', log)
		if newErr != nil {
			log.Warnf("failed to probe user environment variables: %v, %v", err, newErr)
		} else {
			probedEnv = newProbedEnv
		}
	}
	if probedEnv == nil {
		probedEnv = map[string]string{}
	}

	return probedEnv, nil
}

func doProbe(ctx context.Context, userEnvProbe UserEnvProbe, preferredShell []string, userName string, probeCmd string, sep byte, log log.Logger) (map[string]string, error) {
	args := preferredShell
	args = append(args, getShellArgs(userEnvProbe, userName, probeCmd)...)

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, args[0], args[1:]...)

	err := PrepareCmdUser(cmd, userName)
	if err != nil {
		return nil, fmt.Errorf("prepare probe: %w", err)
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("probe user env: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	scanner.Split(splitBySeparator(sep))

	retEnv := map[string]string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		tokens := strings.Split(line, "=")
		if len(tokens) == 1 {
			log.Debugf("failed to split env var: %s", line)
			continue
		}
		retEnv[tokens[0]] = tokens[1]
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("scan shell output: %w", err)
	}
	delete(retEnv, "PWD")

	return retEnv, nil
}

func getShellArgs(userEnvProbe UserEnvProbe, user, command string) []string {
	args := []string{}
	switch userEnvProbe {
	case LoginInteractiveShellProbe:
		args = append(args, "-lic")
	case LoginShellProbe:
		args = append(args, "-lc")
	case InteractiveShellProbe:
		args = append(args, "-ic")
	// shouldn't happen, added just for linting
	case NoneProbe:
		args = append(args, "-c")
	default:
		args = append(args, "-c")
	}
	args = append(args, command)

	return args
}

func splitBySeparator(sep byte) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, sep); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	}
}
