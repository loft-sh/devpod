package setup

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/envfile"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

const (
	ResultLocation = "/var/run/devpod/result.json"
)

func SetupContainer(ctx context.Context, setupInfo *config.Result, extraWorkspaceEnv []string, chownProjects bool, log log.Logger) error {
	// write result to ResultLocation
	WriteResult(setupInfo, log)

	// chown user dir
	err := ChownWorkspace(setupInfo, chownProjects, log)
	if err != nil {
		return errors.Wrap(err, "chown workspace")
	}

	// patch remote env
	log.Debugf("Patch etc environment & profile...")
	err = PatchEtcEnvironment(setupInfo.MergedConfig, log)
	if err != nil {
		return errors.Wrap(err, "patch etc environment")
	}
	err = PatchEtcEnvironmentFlags(extraWorkspaceEnv, log)
	if err != nil {
		return errors.Wrap(err, "patch etc environment from flags")
	}

	// patch etc profile
	err = PatchEtcProfile()
	if err != nil {
		return errors.Wrap(err, "patch etc profile")
	}

	// link /home/root to root if necessary
	err = LinkRootHome(setupInfo)
	if err != nil {
		log.Errorf("Error linking /home/root: %v", err)
	}

	// chown agent sock file
	err = ChownAgentSock(setupInfo)
	if err != nil {
		return errors.Wrap(err, "chown ssh agent sock file")
	}

	// run commands
	log.Debugf("Run lifecycle hooks commands...")
	err = RunLifecycleHooks(ctx, setupInfo, log)
	if err != nil {
		return errors.Wrap(err, "lifecycle hooks")
	}

	log.Debugf("Done setting up environment")
	return nil
}

func WriteResult(setupInfo *config.Result, log log.Logger) {
	rawBytes, err := json.Marshal(setupInfo)
	if err != nil {
		log.Warnf("Error marshal result: %v", err)
		return
	}

	existing, _ := os.ReadFile(ResultLocation)
	if string(rawBytes) == string(existing) {
		return
	}

	err = os.MkdirAll(filepath.Dir(ResultLocation), 0777)
	if err != nil {
		log.Warnf("Error create %s: %v", filepath.Dir(ResultLocation), err)
		return
	}

	err = os.WriteFile(ResultLocation, rawBytes, 0600)
	if err != nil {
		log.Warnf("Error write result to %s: %v", ResultLocation, err)
		return
	}
}

func LinkRootHome(setupInfo *config.Result) error {
	user := config.GetRemoteUser(setupInfo)
	if user != "root" {
		return nil
	}

	home, err := command.GetHome(user)
	if err != nil {
		return errors.Wrap(err, "find root home")
	} else if home == "/home/root" {
		return nil
	}

	_, err = os.Stat("/home/root")
	if err == nil {
		return nil
	}

	// link /home/root to the root home
	err = os.MkdirAll("/home", 0777)
	if err != nil {
		return errors.Wrap(err, "create /home folder")
	}

	err = os.Symlink(home, "/home/root")
	if err != nil {
		return errors.Wrap(err, "create symlink")
	}

	return nil
}

func ChownWorkspace(setupInfo *config.Result, recursive bool, log log.Logger) error {
	user := config.GetRemoteUser(setupInfo)
	exists, err := markerFileExists("chownWorkspace", "")
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	workspaceRoot := filepath.Dir(setupInfo.SubstitutionContext.ContainerWorkspaceFolder)

	if workspaceRoot != "/" {
		log.Infof("Chown workspace...")
		err = copy2.Chown(workspaceRoot, user)
		if err != nil {
			log.Warn(err)
		}
	}

	if recursive {
		log.Infof("Chown projects...")
		err = copy2.ChownR(setupInfo.SubstitutionContext.ContainerWorkspaceFolder, user)
		// do not exit on error, we can have non-fatal errors
		if err != nil {
			log.Warn(err)
		}
	}

	return nil
}

func PatchEtcProfile() error {
	exists, err := markerFileExists("patchEtcProfile", "")
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	out, err := exec.Command("sh", "-c", `sed -i -E 's/((^|\s)PATH=)([^\$]*)$/\1${PATH:-\3}/g' /etc/profile || true`).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "create remote environment: %v", string(out))
	}

	return nil
}

func PatchEtcEnvironmentFlags(workspaceEnv []string, log log.Logger) error {
	if len(workspaceEnv) == 0 {
		return nil
	}

	// make sure we sort the strings
	sort.Strings(workspaceEnv)

	// check if we need to update env
	exists, err := markerFileExists("patchEtcEnvironmentFlags", strings.Join(workspaceEnv, "\n"))
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	// update env
	envfile.MergeAndApply(config.ListToObject(workspaceEnv), log)
	return nil
}

func PatchEtcEnvironment(mergedConfig *config.MergedDevContainerConfig, log log.Logger) error {
	if len(mergedConfig.RemoteEnv) == 0 {
		return nil
	}

	// build remote env
	remoteEnvs := []string{}
	for k, v := range mergedConfig.RemoteEnv {
		remoteEnvs = append(remoteEnvs, k+"=\""+v+"\"")
	}
	sort.Strings(remoteEnvs)

	// check if we need to update env
	exists, err := markerFileExists("patchEtcEnvironment", strings.Join(remoteEnvs, "\n"))
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	// update env
	envfile.MergeAndApply(mergedConfig.RemoteEnv, log)
	return nil
}

func ChownAgentSock(setupInfo *config.Result) error {
	user := config.GetRemoteUser(setupInfo)
	agentSockFile := os.Getenv("SSH_AUTH_SOCK")
	if agentSockFile != "" {
		err := copy2.ChownR(filepath.Dir(agentSockFile), user)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func markerFileExists(markerName string, markerContent string) (bool, error) {
	markerName = filepath.Join("/var/devpod", markerName+".marker")
	t, err := os.ReadFile(markerName)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	} else if err == nil && (markerContent == "" || string(t) == markerContent) {
		return true, nil
	}

	// write marker
	_ = os.MkdirAll(filepath.Dir(markerName), 0777)
	err = os.WriteFile(markerName, []byte(markerContent), 0644)
	if err != nil {
		return false, errors.Wrap(err, "write marker")
	}

	return false, nil
}
