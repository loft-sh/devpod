package setup

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/loft-sh/api/v4/pkg/devpod"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/command"
	copy2 "github.com/loft-sh/devpod/pkg/copy"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/envfile"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	ResultLocation = "/var/run/devpod/result.json"
)

func SetupContainer(ctx context.Context, setupInfo *config.Result, extraWorkspaceEnv []string, chownProjects bool, platformOptions *devpod.PlatformOptions, tunnelClient tunnel.TunnelClient, log log.Logger) error {
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

	// setup kube config
	err = SetupKubeConfig(ctx, setupInfo, tunnelClient, log)
	if err != nil {
		log.Errorf("Error setting up KubeConfig: %v", err)
	}

	// setup platform git credentials
	err = setupPlatformGitCredentials(config.GetRemoteUser(setupInfo), platformOptions, log)
	if err != nil {
		log.Errorf("Error setting up platform git credentials: %v", err)
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

// SetupKubeConfig retrieves and stores a KubeConfig file in the default location `$HOME/.kube/config`.
// It merges our KubeConfig with existing ones.
func SetupKubeConfig(ctx context.Context, setupInfo *config.Result, tunnelClient tunnel.TunnelClient, log log.Logger) error {
	exists, err := markerFileExists("setupKubeConfig", "")
	if err != nil {
		return err
	} else if exists || tunnelClient == nil {
		return nil
	}
	log.Info("Setup KubeConfig")

	// get kubernetes config from setup server
	kubeConfigRes, err := tunnelClient.KubeConfig(ctx, &tunnel.Message{})
	if err != nil {
		return err
	} else if kubeConfigRes.Message == "" {
		return nil
	}

	user := config.GetRemoteUser(setupInfo)
	homeDir, err := command.GetHome(user)
	if err != nil {
		return err
	}

	kubeDir := filepath.Join(homeDir, ".kube")
	err = os.Mkdir(kubeDir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	configPath := filepath.Join(kubeDir, "config")
	existingConfig, err := clientcmd.LoadFromFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if existingConfig == nil {
		existingConfig = clientcmdapi.NewConfig()
	}

	kubeConfig, err := clientcmd.Load([]byte(kubeConfigRes.Message))
	if err != nil {
		return err
	}
	// merge with existing kubeConfig
	for name, cluster := range kubeConfig.Clusters {
		existingConfig.Clusters[name] = cluster
	}
	for name, authInfo := range kubeConfig.AuthInfos {
		existingConfig.AuthInfos[name] = authInfo
	}
	for name, context := range kubeConfig.Contexts {
		existingConfig.Contexts[name] = context
	}

	// Set the current context to the new one.
	// This might not always be the correct choice but given that someone
	// explicitly required this workspace to be in a virtual cluster/space
	// it's fair to assume they also want to point the current context to it
	existingConfig.CurrentContext = kubeConfig.CurrentContext

	err = clientcmd.WriteToFile(*existingConfig, configPath)
	if err != nil {
		return err
	}

	// ensure `remoteUser` owns kubeConfig
	err = copy2.ChownR(kubeDir, user)
	if err != nil {
		return err
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

func setupPlatformGitCredentials(userName string, platformOptions *devpod.PlatformOptions, log log.Logger) error {
	// platform is not enabled, skip
	if !platformOptions.Enabled {
		return nil
	}

	// setup platform git user
	if platformOptions.UserCredentials.GitUser != "" && platformOptions.UserCredentials.GitEmail != "" {
		gitUser, err := gitcredentials.GetUser(userName)
		if err == nil && gitUser.Name == "" && gitUser.Email == "" {
			log.Info("Setup workspace git user and email")
			err := gitcredentials.SetUser(userName, &gitcredentials.GitUser{
				Name:  platformOptions.UserCredentials.GitUser,
				Email: platformOptions.UserCredentials.GitEmail,
			})
			if err != nil {
				return fmt.Errorf("set git user: %w", err)
			}
		}
	}

	// setup platform git http credentials
	err := setupPlatformGitHTTPCredentials(userName, platformOptions, log)
	if err != nil {
		log.Errorf("Error setting up platform git http credentials: %v", err)
	}

	// setup platform git ssh keys
	err = setupPlatformGitSSHKeys(userName, platformOptions, log)
	if err != nil {
		log.Errorf("Error setting up platform git ssh keys: %v", err)
	}

	return nil
}
func setupPlatformGitHTTPCredentials(userName string, platformOptions *devpod.PlatformOptions, log log.Logger) error {
	if !platformOptions.Enabled || len(platformOptions.UserCredentials.GitHttp) == 0 {
		return nil
	}

	log.Info("Setup platform user git http credentials")
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}
	err = gitcredentials.ConfigureHelper(binaryPath, userName, -1)
	if err != nil {
		return fmt.Errorf("configure git helper: %w", err)
	}

	return nil
}

func setupPlatformGitSSHKeys(userName string, platformOptions *devpod.PlatformOptions, log log.Logger) error {
	if !platformOptions.Enabled || len(platformOptions.UserCredentials.GitSsh) == 0 {
		return nil
	}

	log.Info("Setup platform user git ssh keys")
	homeFolder, err := command.GetHome(userName)
	if err != nil {
		return err
	}

	// write ssh keys to ~/.ssh/id_rsa
	sshFolder := filepath.Join(homeFolder, ".ssh")
	err = os.MkdirAll(sshFolder, 0700)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	_ = copy2.Chown(sshFolder, userName)

	// delete previous keys
	files, err := os.ReadDir(sshFolder)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "platform_git_ssh_") {
			continue
		}

		fileName := strings.TrimPrefix(file.Name(), "platform_git_ssh_")
		index, err := strconv.Atoi(fileName)
		if err != nil {
			continue
		}
		if index >= len(platformOptions.UserCredentials.GitSsh) {
			continue
		}

		err = os.Remove(filepath.Join(sshFolder, file.Name()))
		if err != nil {
			log.Warnf("Error removing previous platform git ssh key: %v", err)
		}
	}

	// write new keys
	for i, key := range platformOptions.UserCredentials.GitSsh {
		fileName := filepath.Join(sshFolder, fmt.Sprintf("platform_git_ssh_%d", i))

		// base64 decode before writing to file
		decoded, err := base64.StdEncoding.DecodeString(key.Key)
		if err != nil {
			log.Warnf("Error decoding platform git ssh key: %v", err)
			continue
		}
		err = os.WriteFile(fileName, decoded, 0600)
		if err != nil {
			log.Warnf("Error writing platform git ssh key: %v", err)
			continue
		}

		err = copy2.Chown(fileName, userName)
		// do not exit on error, we can have non-fatal errors
		if err != nil {
			log.Warnf("Error chowning platform git ssh keys: %v", err)
		}
	}

	return nil
}
