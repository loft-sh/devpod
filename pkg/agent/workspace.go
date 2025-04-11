package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/loft-sh/api/v4/pkg/devpod"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/util"
	"github.com/loft-sh/log"
	"github.com/moby/patternmatcher/ignorefile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var extraSearchLocations = []string{"/home/devpod/.devpod/agent", "/opt/devpod/agent", "/var/lib/devpod/agent", "/var/devpod/agent"}

var ErrFindAgentHomeFolder = fmt.Errorf("couldn't find devpod home directory")

func GetAgentDaemonLogFolder(agentFolder string) (string, error) {
	return FindAgentHomeFolder(agentFolder)
}

func findDir(agentFolder string, validate func(path string) bool) string {
	// get agent folder
	if agentFolder != "" {
		if !validate(agentFolder) {
			return ""
		}

		return agentFolder
	}

	// check environment
	homeFolder := os.Getenv(config.DEVPOD_HOME)
	if homeFolder != "" {
		homeFolder = filepath.Join(homeFolder, "agent")
		if !validate(homeFolder) {
			return ""
		}

		return homeFolder
	}

	// check home folder first
	homeDir, _ := util.UserHomeDir()
	if homeDir != "" {
		homeDir = filepath.Join(homeDir, ".devpod", "agent")
		if validate(homeDir) {
			return homeDir
		}
	}

	// check root folder
	homeDir, _ = command.GetHome("root")
	if homeDir != "" {
		homeDir = filepath.Join(homeDir, ".devpod", "agent")
		if validate(homeDir) {
			return homeDir
		}
	}

	// check current directory
	execDir, _ := os.Executable()
	if execDir != "" {
		execDir = filepath.Join(filepath.Dir(execDir), "agent")
		if validate(execDir) {
			return execDir
		}
	}

	// check other folders
	for _, dir := range extraSearchLocations {
		if validate(dir) {
			return dir
		}
	}

	return ""
}

func FindAgentHomeFolder(agentFolder string) (string, error) {
	homeDir := findDir(agentFolder, isDevPodHome)
	if homeDir != "" {
		return homeDir, nil
	}

	return "", ErrFindAgentHomeFolder
}

func isDevPodHome(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "contexts"))
	return err == nil
}

func PrepareAgentHomeFolder(agentFolder string) (string, error) {
	// try to find agent home folder first
	homeFolder, err := FindAgentHomeFolder(agentFolder)
	if err == nil {
		return homeFolder, nil
	}

	// try to find an executable directory
	homeDir := findDir(agentFolder, func(path string) bool {
		ok, _ := isDirExecutable(path)
		return ok
	})
	if homeDir != "" {
		return homeDir, nil
	}

	// check if agentFolder is set
	if agentFolder != "" {
		_, err := isDirExecutable(agentFolder)
		return "", err
	}

	// return generic error here
	return "", fmt.Errorf("couldn't find an executable directory")
}

func isDirExecutable(dir string) (bool, error) {
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return false, err
		}
	}

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return false, err
	}

	testFile := filepath.Join(dir, "devpod_test.sh")
	err = os.WriteFile(testFile, []byte(`#!/bin/sh
echo DevPod
`), 0755)
	if err != nil {
		return false, err
	}
	defer os.Remove(testFile)
	if runtime.GOOS != "linux" {
		return true, nil
	}

	// try to execute
	out, err := exec.Command(testFile).Output()
	if err != nil {
		return false, err
	} else if strings.TrimSpace(string(out)) != "DevPod" {
		return false, fmt.Errorf("received %s, expected DevPod", strings.TrimSpace(string(out)))
	}

	return true, nil
}

func GetAgentWorkspaceContentDir(workspaceDir string) string {
	return filepath.Join(workspaceDir, "content")
}

func GetAgentBinariesDirFromWorkspaceDir(workspaceDir string) (string, error) {
	// check if it already exists
	_, err := os.Stat(workspaceDir)
	if err == nil {
		return filepath.Join(workspaceDir, "binaries"), nil
	}

	return "", os.ErrNotExist
}

func GetAgentBinariesDir(agentFolder, context, workspaceID string) (string, error) {
	homeFolder, err := FindAgentHomeFolder(agentFolder)
	if err != nil {
		return "", err
	}
	if context == "" {
		context = config.DefaultContext
	}

	// workspace folder
	workspaceDir := filepath.Join(homeFolder, "contexts", context, "workspaces", workspaceID)

	// get from workspace folder
	return GetAgentBinariesDirFromWorkspaceDir(workspaceDir)
}

func GetAgentWorkspaceDir(agentFolder, context, workspaceID string) (string, error) {
	homeFolder, err := FindAgentHomeFolder(agentFolder)
	if err != nil {
		return "", err
	}
	if context == "" {
		context = config.DefaultContext
	}

	// workspace folder
	workspaceDir := filepath.Join(homeFolder, "contexts", context, "workspaces", workspaceID)

	// check if it already exists
	_, err = os.Stat(workspaceDir)
	if err == nil {
		return workspaceDir, nil
	}

	return "", os.ErrNotExist
}

func CreateAgentWorkspaceDir(agentFolder, context, workspaceID string) (string, error) {
	homeFolder, err := PrepareAgentHomeFolder(agentFolder)
	if err != nil {
		return "", err
	}

	// workspace folder
	workspaceDir := filepath.Join(homeFolder, "contexts", context, "workspaces", workspaceID)

	// create workspace folder
	err = os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		return "", err
	}

	return workspaceDir, nil
}

func CloneRepositoryForWorkspace(
	ctx context.Context,
	source *provider2.WorkspaceSource,
	agentConfig *provider2.ProviderAgentConfig,
	workspaceDir, helper string,
	options provider2.CLIOptions,
	overwriteContent bool,
	log log.Logger,
) error {
	log.Info("Clone repository")
	log.Infof("URL: %s\n", source.GitRepository)
	if source.GitBranch != "" {
		log.Infof("Branch: %s\n", source.GitBranch)
	}
	if source.GitCommit != "" {
		log.Infof("Commit: %s\n", source.GitCommit)
	}
	if source.GitSubPath != "" {
		log.Infof("Subpath: %s\n", source.GitSubPath)
	}
	if source.GitPRReference != "" {
		log.Infof("PR: %s\n", source.GitPRReference)
	}

	// remove the credential helper or otherwise we will receive strange errors within the container
	defer func() {
		if helper != "" {
			if err := gitcredentials.RemoveHelperFromPath(gitcredentials.GetLocalGitConfigPath(workspaceDir)); err != nil {
				log.Errorf("Remove git credential helper: %v", err)
			}
		}
	}()

	// check if command exists
	if !command.Exists("git") {
		local, _ := agentConfig.Local.Bool()
		if local {
			return fmt.Errorf("seems like git isn't installed on your system. Please make sure to install git and make it available in the PATH")
		}
		if err := git.InstallBinary(log); err != nil {
			return err
		}
	}

	if overwriteContent {
		if err := removeDirContents(workspaceDir); err != nil {
			log.Infof("Failed cleanup")
			return err
		}
	}

	// setup private ssh key if passed in
	extraEnv := []string{}
	gitSshCredentials := append(options.Platform.UserCredentials.GitSsh, options.Platform.ProjectCredentials.GitSsh...)
	if len(gitSshCredentials) > 0 {
		keys := []string{}
		for _, key := range gitSshCredentials {
			keys = append(keys, key.Key)
		}

		sshExtraEnv, cleanUpSSHKey, err := setupSSHKey(keys, agentConfig.Path)
		if err != nil {
			return err
		}
		defer cleanUpSSHKey()
		extraEnv = append(extraEnv, sshExtraEnv...)
	}

	// run git command
	gitInfo := git.NewGitInfo(source.GitRepository, source.GitBranch, source.GitCommit, source.GitPRReference, source.GitSubPath)

	// should run with platform git cache?
	platformGitcacheEnabled := options.Platform.Enabled && options.Platform.RunnerSocket != ""
	if platformGitcacheEnabled {
		_, err := os.Stat(options.Platform.RunnerSocket)
		if err != nil {
			platformGitcacheEnabled = false
		}
	}

	// try to clone with platform gitcache
	if platformGitcacheEnabled {
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, "unix", options.Platform.RunnerSocket)
		if err != nil {
			return fmt.Errorf("dial platform gitcache: %w", err)
		}
		defer conn.Close()

		// Set up a connection to the server.
		grpcClient, err := grpc.NewClient(
			"unix://"+options.Platform.RunnerSocket,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithIdleTimeout(180*time.Minute), // cloning can take a long time for large monorepos
		)
		if err != nil {
			return fmt.Errorf("create platform gitcache client: %w", err)
		}

		// marshal options
		jsonOptions, err := json.Marshal(&devpod.CloneOptions{
			Repository:        source.GitRepository,
			Branch:            source.GitBranch,
			Commit:            source.GitCommit,
			PRReference:       source.GitPRReference,
			SubPath:           source.GitSubPath,
			CredentialsHelper: helper,
			ExtraEnv:          append(git.GetDefaultExtraEnv(options.StrictHostKeyChecking), extraEnv...),
		})
		if err != nil {
			return fmt.Errorf("marshal git options: %w", err)
		}

		// create client
		log.Infof("Cloning repository %s in platform", source.GitRepository)
		_, err = devpod.NewRunnerClient(grpcClient).Clone(ctx, &devpod.CloneRequest{
			TargetPath: workspaceDir,
			Options:    string(jsonOptions),
		})
		if err != nil {
			// unpack error
			statusErr, ok := status.FromError(err)
			if ok && statusErr.Message() != "" {
				err = errors.New(statusErr.Message())
			}

			// cleanup workspace dir if clone failed, otherwise we won't try to clone again when rebuilding this workspace
			if cleanupErr := cleanupWorkspaceDir(workspaceDir); cleanupErr != nil {
				return fmt.Errorf("clone repository (with gitcache): %w, cleanup workspace: %w", err, cleanupErr)
			}
			return fmt.Errorf("clone repository (with gitcache): %w", err)
		}
	} else {
		if options.Platform.GitCloneStrategy != "" {
			log.Infof("Using a %s clone", options.Platform.GitCloneStrategy)
		}
		if options.Platform.GitSkipLFS {
			log.Info("Skipping Git LFS")
		}
		err := git.CloneRepositoryWithEnv(ctx, gitInfo, extraEnv, workspaceDir, helper, options.StrictHostKeyChecking, log, getGitOptions(options)...)
		if err != nil {
			// cleanup workspace dir if clone failed, otherwise we won't try to clone again when rebuilding this workspace
			if cleanupErr := cleanupWorkspaceDir(workspaceDir); cleanupErr != nil {
				return fmt.Errorf("clone repository: %w, cleanup workspace: %w", err, cleanupErr)
			}
			return fmt.Errorf("clone repository: %w", err)
		}
	}

	log.Done("Successfully cloned repository")

	// Get .devpodignore files to exclude
	f, err := os.Open(filepath.Join(workspaceDir, ".devpodignore"))
	if err != nil {
		return nil
	}
	excludes, err := ignorefile.ReadAll(f)
	if err != nil {
		log.Warn(".devpodignore file is invalid : ", err)
		return nil
	}
	// Remove files from workspace content folder
	for _, exclude := range excludes {
		os.RemoveAll(filepath.Join(workspaceDir, exclude))
	}
	log.Debug("Ignore files from .devpodignore ", excludes)

	return nil
}

func getGitOptions(options provider2.CLIOptions) []git.Option {
	gitOpts := []git.Option{}
	if options.GitCloneStrategy != "" {
		gitOpts = append(gitOpts, git.WithCloneStrategy(options.GitCloneStrategy))
	}
	if options.Platform.GitCloneStrategy != "" {
		gitOpts = append(gitOpts, git.WithCloneStrategy(git.CloneStrategy(options.Platform.GitCloneStrategy)))
	}
	if options.Platform.GitSkipLFS {
		gitOpts = append(gitOpts, git.WithSkipLFS())
	}
	if options.GitCloneRecursiveSubmodules {
		gitOpts = append(gitOpts, git.WithRecursiveSubmodules())
	}
	return gitOpts
}

func cleanupWorkspaceDir(workspaceDir string) error {
	return os.RemoveAll(workspaceDir)
}

func setupSSHKey(keys []string, agentPath string) ([]string, func(), error) {
	keyFiles := []string{}
	for _, key := range keys {
		keyFile, err := os.CreateTemp("", "")
		if err != nil {
			return nil, nil, err
		}
		defer keyFile.Close()

		if err := writeSSHKey(keyFile, key); err != nil {
			return nil, nil, err
		}

		if err := os.Chmod(keyFile.Name(), 0o400); err != nil {
			return nil, nil, err
		}

		keyFiles = append(keyFiles, keyFile.Name())
	}

	env := []string{"GIT_TERMINAL_PROMPT=0"}
	gitSSHCmd := []string{agentPath, "helper", "ssh-git-clone"}
	for _, keyFile := range keyFiles {
		gitSSHCmd = append(gitSSHCmd, "--key-file="+keyFile)
	}

	env = append(env, "GIT_SSH_COMMAND="+command.Quote(gitSSHCmd))
	cleanup := func() {
		for _, keyFile := range keyFiles {
			os.Remove(keyFile)
		}
	}

	return env, cleanup, nil
}

func writeSSHKey(key *os.File, sshKey string) error {
	data, err := base64.StdEncoding.DecodeString(sshKey)
	if err != nil {
		return err
	}

	_, err = key.WriteString(string(data))
	return err
}

func removeDirContents(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			err = os.RemoveAll(entryPath)
		} else {
			err = os.Remove(entryPath)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
