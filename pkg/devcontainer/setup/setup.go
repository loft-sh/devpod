package setup

import (
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

func SetupContainer(setupInfo *config.Result) error {
	// chown user dir
	err := ChownWorkspace(setupInfo.ContainerDetails, setupInfo.MergedConfig, setupInfo.SubstitutionContext)
	if err != nil {
		return errors.Wrap(err, "chown workspace")
	}

	// patch remote env
	err = PatchEtcEnvironment(setupInfo.MergedConfig)
	if err != nil {
		return errors.Wrap(err, "patch etc environment")
	}

	// patch etc profile
	err = PatchEtcProfile()
	if err != nil {
		return errors.Wrap(err, "patch etc profile")
	}

	return nil
}

func ChownWorkspace(containerDetails *config.ContainerDetails, mergedConfig *config.MergedDevContainerConfig, substitutionContext *config.SubstitutionContext) error {
	user := mergedConfig.RemoteUser
	if mergedConfig.RemoteUser == "" {
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

func PostCreateCommands() error {
	return nil
}
