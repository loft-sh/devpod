package setup

import (
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func SetupContainer(setupInfo *config.Result, log log.Logger) error {
	// chown user dir
	log.Debugf("Chown workspace...")
	err := ChownWorkspace(setupInfo.ContainerDetails, setupInfo.MergedConfig, setupInfo.SubstitutionContext)
	if err != nil {
		return errors.Wrap(err, "chown workspace")
	}

	// patch remote env
	log.Debugf("Patch etc environment & profile...")
	err = PatchEtcEnvironment(setupInfo.MergedConfig)
	if err != nil {
		return errors.Wrap(err, "patch etc environment")
	}

	// patch etc profile
	err = PatchEtcProfile()
	if err != nil {
		return errors.Wrap(err, "patch etc profile")
	}

	// run commands
	log.Debugf("Run post create commands...")
	err = PostCreateCommands(setupInfo, log)
	if err != nil {
		return errors.Wrap(err, "post create commands")
	}

	log.Debugf("Done setting up environment")
	return nil
}

func ChownWorkspace(containerDetails *config.ContainerDetails, mergedConfig *config.MergedDevContainerConfig, substitutionContext *config.SubstitutionContext) error {
	user := mergedConfig.RemoteUser
	if mergedConfig.RemoteUser == "" && containerDetails != nil {
		user = containerDetails.Config.User
	}
	if user == "" {
		user = "root"
	}

	_, err := exec.Command("sh", "-c", "ls /var/devcontainer/.chownWorkspace").CombinedOutput()
	if err == nil {
		return nil
	}

	out, err := exec.Command("sh", "-c", "mkdir -p /var/devcontainer && touch /var/devcontainer/.chownWorkspace").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "create marker file: %v", string(out))
	}

	out, err = exec.Command("sh", "-c", `chown -R `+user+` `+substitutionContext.ContainerWorkspaceFolder).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "chown workspace folder: %v", string(out))
	}

	// make sure all volume mounts are owned by the correct user
	for _, mount := range mergedConfig.Mounts {
		if mount.Type != "volume" || mount.Target == "" {
			continue
		}

		out, err = exec.Command("sh", "-c", `chown -R `+user+` `+mount.Target).CombinedOutput()
		if err != nil {
			return errors.Wrapf(err, "chown volume: %v", string(out))
		}
	}

	return nil
}

func PatchEtcProfile() error {
	_, err := exec.Command("sh", "-c", "ls /var/devcontainer/.patchEtcProfileMarker").CombinedOutput()
	if err == nil {
		return nil
	}

	out, err := exec.Command("sh", "-c", "mkdir -p /var/devcontainer && touch /var/devcontainer/.patchEtcProfileMarker").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "create marker file: %v", string(out))
	}

	out, err = exec.Command("sh", "-c", `sed -i -E 's/((^|\s)PATH=)([^\$]*)$/\1${PATH:-\3}/g' /etc/profile || true`).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "create remote environment: %v", string(out))
	}

	return nil
}

func PatchEtcEnvironment(mergedConfig *config.MergedDevContainerConfig) error {
	if len(mergedConfig.RemoteEnv) == 0 {
		return nil
	}

	_, err := exec.Command("sh", "-c", "ls /var/devcontainer/.patchEtcEnvironmentMarker").CombinedOutput()
	if err == nil {
		return nil
	}

	out, err := exec.Command("sh", "-c", "mkdir -p /var/devcontainer && touch /var/devcontainer/.patchEtcEnvironmentMarker").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "create marker file: %v", string(out))
	}

	// build remote env
	remoteEnvs := []string{}
	for k, v := range mergedConfig.RemoteEnv {
		remoteEnvs = append(remoteEnvs, k+"=\""+v+"\"")
	}

	out, err = exec.Command("sh", "-c", `cat >> /etc/environment <<'etcEnvrionmentEOF'
`+strings.Join(remoteEnvs, "\n")+`
etcEnvrionmentEOF
`).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "create remote environment: %v", string(out))
	}

	return nil
}

func PostCreateCommands(setupInfo *config.Result, log log.Logger) error {
	remoteUser := config.GetRemoteUser(setupInfo)
	mergedConfig := setupInfo.MergedConfig

	// only run once per container run
	err := runPostCreateCommand(mergedConfig.OnCreateCommands, remoteUser, setupInfo.SubstitutionContext.ContainerWorkspaceFolder, setupInfo.MergedConfig.RemoteEnv, "onCreateCommands", setupInfo.ContainerDetails.Created, log)
	if err != nil {
		return err
	}

	//TODO: rerun when contents changed
	err = runPostCreateCommand(mergedConfig.UpdateContentCommands, remoteUser, setupInfo.SubstitutionContext.ContainerWorkspaceFolder, setupInfo.MergedConfig.RemoteEnv, "updateContentCommands", setupInfo.ContainerDetails.Created, log)
	if err != nil {
		return err
	}

	// only run once per container run
	err = runPostCreateCommand(mergedConfig.PostCreateCommands, remoteUser, setupInfo.SubstitutionContext.ContainerWorkspaceFolder, setupInfo.MergedConfig.RemoteEnv, "postCreateCommands", setupInfo.ContainerDetails.Created, log)
	if err != nil {
		return err
	}

	// run when the container was restarted
	err = runPostCreateCommand(mergedConfig.PostStartCommands, remoteUser, setupInfo.SubstitutionContext.ContainerWorkspaceFolder, setupInfo.MergedConfig.RemoteEnv, "postStartCommands", setupInfo.ContainerDetails.State.StartedAt, log)
	if err != nil {
		return err
	}

	// run always when attaching to the container
	err = runPostCreateCommand(mergedConfig.PostAttachCommands, remoteUser, setupInfo.SubstitutionContext.ContainerWorkspaceFolder, setupInfo.MergedConfig.RemoteEnv, "postAttachCommands", "", log)
	if err != nil {
		return err
	}

	return nil
}

func runPostCreateCommand(commands []types.StrArray, user, dir string, remoteEnv map[string]string, name, content string, log log.Logger) error {
	if len(commands) == 0 {
		return nil
	}

	// check marker file
	if content != "" {
		homeDir, err := command.GetHome(user)
		if err != nil {
			return errors.Wrap(err, "find user home")
		}

		markerName := filepath.Join(homeDir, ".devpod", name+".marker")
		t, err := os.ReadFile(markerName)
		if err != nil && !os.IsNotExist(err) {
			return err
		} else if string(t) == content {
			return nil
		}

		// write marker
		_ = os.MkdirAll(filepath.Dir(markerName), 0777)
		err = os.WriteFile(markerName, []byte(content), 0666)
		if err != nil {
			return errors.Wrap(err, "write marker")
		}
	}

	remoteEnvArr := []string{}
	for k, v := range remoteEnv {
		remoteEnvArr = append(remoteEnvArr, k+"="+v)
	}

	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	for _, c := range commands {
		log.Infof("Run command: %s", strings.Join(c, " "))
		args := []string{}
		if user != "root" {
			args = append(args, "su", user, "-c", strings.Join(c, " "))
		} else {
			args = append(args, c...)
		}

		// create command
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, remoteEnvArr...)
		cmd.Stdout = writer
		cmd.Stderr = writer
		err := cmd.Run()
		if err != nil {
			log.Errorf("Failed running command: %v", err)
			return err
		}
		log.Infof("Successfully ran command: %s", strings.Join(c, " "))
	}

	return nil
}
