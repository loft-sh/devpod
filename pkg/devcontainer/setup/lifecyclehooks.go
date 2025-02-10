package setup

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
)

func RunLifecycleHooks(ctx context.Context, setupInfo *config.Result, log log.Logger) error {
	mergedConfig := setupInfo.MergedConfig
	remoteUser := config.GetRemoteUser(setupInfo)
	probedEnv, err := config.ProbeUserEnv(ctx, mergedConfig.UserEnvProbe, remoteUser, log)
	if err != nil {
		log.Errorf("failed to probe environment, this might lead to an incomplete setup of your workspace: %w", err)
	}
	remoteEnv := mergeRemoteEnv(mergedConfig.RemoteEnv, probedEnv, remoteUser)

	workspaceFolder := setupInfo.SubstitutionContext.ContainerWorkspaceFolder
	containerDetails := setupInfo.ContainerDetails

	// only run once per container run
	err = run(mergedConfig.OnCreateCommands, remoteUser, workspaceFolder, remoteEnv,
		"onCreateCommands", containerDetails.Created, log)
	if err != nil {
		return err
	}

	// TODO: rerun when contents changed
	err = run(mergedConfig.UpdateContentCommands, remoteUser, workspaceFolder, remoteEnv,
		"updateContentCommands", containerDetails.Created, log)
	if err != nil {
		return err
	}

	// only run once per container run
	err = run(mergedConfig.PostCreateCommands, remoteUser, workspaceFolder, remoteEnv,
		"postCreateCommands", containerDetails.Created, log)
	if err != nil {
		return err
	}

	// run when the container was restarted
	err = run(mergedConfig.PostStartCommands, remoteUser, workspaceFolder, remoteEnv,
		"postStartCommands", containerDetails.State.StartedAt, log)
	if err != nil {
		return err
	}

	// run always when attaching to the container
	err = run(mergedConfig.PostAttachCommands, remoteUser, workspaceFolder, remoteEnv,
		"postAttachCommands", "", log)
	if err != nil {
		return err
	}

	return nil
}

func run(commands []types.LifecycleHook, remoteUser, dir string, remoteEnv map[string]string, name, content string, log log.Logger) error {
	if len(commands) == 0 {
		return nil
	}

	// check marker file
	if content != "" {
		exists, err := markerFileExists(name, content)
		if err != nil {
			return err
		} else if exists {
			return nil
		}
	}

	remoteEnvArr := []string{}
	for k, v := range remoteEnv {
		remoteEnvArr = append(remoteEnvArr, k+"="+v)
	}

	for _, cmd := range commands {
		if len(cmd) == 0 {
			continue
		}

		for k, c := range cmd {
			log.Infof("Run command %s: %s...", k, strings.Join(c, " "))
			currentUser, err := user.Current()
			if err != nil {
				return err
			}
			args := []string{}
			if remoteUser != currentUser.Username {
				args = append(args, "su", remoteUser, "-c", command.Quote(c))
			} else {
				args = append(args, "sh", "-c", command.Quote(c))
			}

			// create command
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = dir
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, remoteEnvArr...)

			// Create pipes for stdout and stderr
			stdoutPipe, err := cmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("failed to get stdout pipe: %w", err)
			}
			stderrPipe, err := cmd.StderrPipe()
			if err != nil {
				return fmt.Errorf("failed to get stderr pipe: %w", err)
			}

			// Start the command
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start command: %w", err)
			}

			// Use WaitGroup to wait for both stdout and stderr processing
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				logPipeOutput(log, stdoutPipe, logrus.InfoLevel)
			}()

			go func() {
				defer wg.Done()
				logPipeOutput(log, stderrPipe, logrus.ErrorLevel)
			}()

			// Wait for command to finish
			wg.Wait()
			err = cmd.Wait()
			if err != nil {
				log.Debugf("Failed running postCreateCommand lifecycle script %s: %v", cmd.Args, err)
				return fmt.Errorf("failed to run: %s, error: %w", strings.Join(c, " "), err)
			}

			log.Donef("Successfully ran command %s: %s", k, strings.Join(c, " "))
		}
	}

	return nil
}

func logPipeOutput(log log.Logger, pipe io.ReadCloser, level logrus.Level) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		if level == logrus.InfoLevel {
			log.Info(line)
		} else if level == logrus.ErrorLevel {
			if containsError(line) {
				log.Error(line)
			} else {
				log.Warn(line)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Errorf("Error reading pipe: %v", err)
	}
}

// containsError defines what log line treated as error log should contain.
func containsError(line string) bool {
	return strings.Contains(strings.ToLower(line), "error")
}

func mergeRemoteEnv(remoteEnv map[string]string, probedEnv map[string]string, remoteUser string) map[string]string {
	retEnv := map[string]string{}

	// Order matters here
	// remoteEnv should always override probedEnv as it has been specified explicitly by the devcontainer author
	for k, v := range probedEnv {
		retEnv[k] = v
	}
	for k, v := range remoteEnv {
		retEnv[k] = v
	}
	probedPath, probeOk := probedEnv["PATH"]
	remotePath, remoteOk := remoteEnv["PATH"]
	if probeOk && remoteOk {
		// merge probed PATH and remote PATH
		sbinRegex := regexp.MustCompile(`/sbin(/|$)`)
		probedTokens := strings.Split(probedPath, ":")
		insertAt := 0
		for _, e := range strings.Split(remotePath, ":") {
			// check if remotePath entry is in probed tokens
			i := slices.Index(probedTokens, e)
			if i == -1 {
				// only include /sbin paths for root users
				if remoteUser == "root" || !sbinRegex.MatchString(e) {
					probedTokens = slices.Insert(probedTokens, insertAt, e)
				}
			} else {
				insertAt = i + 1
			}
		}

		retEnv["PATH"] = strings.Join(probedTokens, ":")
	}

	return retEnv
}
