package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/agent/tunnelserver"
	"github.com/loft-sh/devpod/pkg/binaries"
	"github.com/loft-sh/devpod/pkg/client/clientimplementation"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/credentials"
	"github.com/loft-sh/devpod/pkg/daemon"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/dockercredentials"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/scripts"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	*flags.GlobalFlags

	WorkspaceInfo string
}

// NewUpCmd creates a new command
func NewUpCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: flags,
	}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts a new devcontainer",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	upCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = upCmd.MarkFlagRequired("workspace-info")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context) error {
	// get workspace
	shouldExit, workspaceInfo, err := agent.WriteWorkspaceInfoAndDeleteOld(cmd.WorkspaceInfo, func(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
		return deleteWorkspace(ctx, workspaceInfo, log)
	}, log.Default.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error parsing workspace info: %w", err)
	} else if shouldExit {
		return nil
	}

	// make sure daemon doesn't shut us down while we are doing things
	agent.CreateWorkspaceBusyFile(workspaceInfo.Origin)
	defer agent.DeleteWorkspaceBusyFile(workspaceInfo.Origin)

	// initialize the workspace
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	tunnelClient, logger, credentialsDir, err := initWorkspace(cancelCtx, cancel, workspaceInfo, cmd.Debug, !workspaceInfo.CLIOptions.Proxy && !workspaceInfo.CLIOptions.DisableDaemon)
	if err != nil {
		err1 := clientimplementation.DeleteWorkspaceFolder(workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID, workspaceInfo.Workspace.SSHConfigPath, logger)
		if err1 != nil {
			return errors.Wrap(err, err1.Error())
		}
		return err
	} else if credentialsDir != "" {
		defer func() {
			_ = os.RemoveAll(credentialsDir)
		}()
	}

	// start up
	err = cmd.up(ctx, workspaceInfo, tunnelClient, logger)
	if err != nil {
		return errors.Wrap(err, "devcontainer up")
	}

	return nil
}

func (cmd *UpCmd) up(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, tunnelClient tunnel.TunnelClient, logger log.Logger) error {
	// create devcontainer
	result, err := cmd.devPodUp(ctx, workspaceInfo, logger)
	if err != nil {
		return err
	}

	// send result
	out, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tunnelClient.SendResult(ctx, &tunnel.Message{Message: string(out)})
	if err != nil {
		return errors.Wrap(err, "send result")
	}

	return nil
}

func (cmd *UpCmd) devPodUp(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (*config2.Result, error) {
	runner, err := CreateRunner(workspaceInfo, log)
	if err != nil {
		return nil, err
	}

	// start the devcontainer
	result, err := runner.Up(ctx, devcontainer.UpOptions{
		CLIOptions: workspaceInfo.CLIOptions,
	}, workspaceInfo.InjectTimeout)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CreateRunner(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (devcontainer.Runner, error) {
	return devcontainer.NewRunner(agent.ContainerDevPodHelperLocation, agent.DefaultAgentDownloadURL(), workspaceInfo, log)
}

func InitContentFolder(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (bool, error) {
	// check if workspace content folder exists
	_, err := os.Stat(workspaceInfo.ContentFolder)
	if err == nil {
		log.Debugf("Workspace Folder already exists %s", workspaceInfo.ContentFolder)
		return true, nil
	}

	// make content dir
	log.Debugf("Create content folder %s", workspaceInfo.ContentFolder)
	err = os.MkdirAll(workspaceInfo.ContentFolder, 0o777)
	if err != nil {
		return false, errors.Wrap(err, "make workspace folder")
	}

	// download provider
	binariesDir, err := agent.GetAgentBinariesDir(workspaceInfo.Agent.DataPath, workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID)
	if err != nil {
		_ = os.RemoveAll(workspaceInfo.ContentFolder)
		return false, fmt.Errorf("error getting workspace %s binaries dir: %w", workspaceInfo.Workspace.ID, err)
	}

	// download binaries
	_, err = binaries.DownloadBinaries(workspaceInfo.Agent.Binaries, binariesDir, log)
	if err != nil {
		_ = os.RemoveAll(workspaceInfo.ContentFolder)
		return false, fmt.Errorf("error downloading workspace %s binaries: %w", workspaceInfo.Workspace.ID, err)
	}

	// if workspace was already executed, we skip this part
	if workspaceInfo.LastDevContainerConfig != nil {
		// make sure the devcontainer.json exists
		err = ensureLastDevContainerJson(workspaceInfo)
		if err != nil {
			log.Errorf("Ensure devcontainer.json: %v", err)
		}

		return true, nil
	}

	return false, nil
}

func initWorkspace(ctx context.Context, cancel context.CancelFunc, workspaceInfo *provider2.AgentWorkspaceInfo, debug, shouldInstallDaemon bool) (tunnel.TunnelClient, log.Logger, string, error) {
	// create a grpc client
	tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true, 0)
	if err != nil {
		return nil, nil, "", fmt.Errorf("error creating tunnel client: %w", err)
	}

	// create debug logger
	logger := tunnelserver.NewTunnelLogger(ctx, tunnelClient, debug)
	logger.Debugf("Created logger")

	// this message serves as a ping to the client
	_, err = tunnelClient.Ping(ctx, &tunnel.Empty{})
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "ping client")
	}

	// get docker credentials
	dockerCredentialsDir, gitCredentialsHelper, err := configureCredentials(ctx, cancel, workspaceInfo, tunnelClient, logger)
	if err != nil {
		logger.Errorf("Error retrieving docker / git credentials: %v", err)
	}

	// install docker in background
	errChan := make(chan error, 1)
	go func() {
		if !workspaceInfo.Agent.IsDockerDriver() || workspaceInfo.Agent.Docker.Install == "false" {
			errChan <- nil
		} else {
			errChan <- installDocker(logger)
		}
	}()

	// prepare workspace
	err = prepareWorkspace(ctx, workspaceInfo, tunnelClient, gitCredentialsHelper, logger)
	if err != nil {
		return nil, logger, "", err
	}

	// install daemon
	if shouldInstallDaemon {
		err = installDaemon(workspaceInfo, logger)
		if err != nil {
			logger.Errorf("Install DevPod Daemon: %v", err)
		}
	}

	// wait until docker is installed
	err = <-errChan
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "install docker")
	}

	return tunnelClient, logger, dockerCredentialsDir, nil
}

func prepareWorkspace(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, client tunnel.TunnelClient, helper string, log log.Logger) error {
	// change content folder if source is local folder in proxy mode
	// to a folder that's known ahead of time inside of DEVPOD_HOME
	if workspaceInfo.CLIOptions.Proxy && workspaceInfo.Workspace.Source.LocalFolder != "" {
		workspaceInfo.ContentFolder = agent.GetAgentWorkspaceContentDir(workspaceInfo.Origin)
	}

	// make sure content folder exists
	exists, err := InitContentFolder(workspaceInfo, log)
	if err != nil {
		return err
	} else if exists && !workspaceInfo.CLIOptions.Recreate {
		log.Debugf("Workspace exists, skip downloading")
		return nil
	}

	// check what type of workspace this is
	if workspaceInfo.Workspace.Source.GitRepository != "" {
		if workspaceInfo.CLIOptions.Reset {
			log.Info("Resetting git based workspace, removing old content folder")
			err = os.RemoveAll(workspaceInfo.ContentFolder)
			if err != nil {
				log.Warnf("Failed to remove workspace folder, still proceeding: %v", err)
			}
		}

		if workspaceInfo.CLIOptions.Recreate && !workspaceInfo.CLIOptions.Reset {
			log.Info("Rebuiling without resetting a git based workspace, keeping old content folder")
			return nil
		}

		return cloneRepository(ctx, workspaceInfo, helper, log)
	}

	if workspaceInfo.Workspace.Source.LocalFolder != "" {
		// if we're not sending this to a remote machine, we're already done
		if workspaceInfo.ContentFolder == workspaceInfo.Workspace.Source.LocalFolder {
			log.Debugf("Local folder with local provider; skip downloading")
			return nil
		}

		log.Debugf("Download Local Folder")
		return downloadLocalFolder(ctx, workspaceInfo.ContentFolder, client, log)
	}

	if workspaceInfo.Workspace.Source.Image != "" {
		log.Debugf("Prepare Image")
		return prepareImage(workspaceInfo.ContentFolder, workspaceInfo.Workspace.Source.Image)
	}

	if workspaceInfo.Workspace.Source.Container != "" {
		log.Debugf("Workspace is a container, nothing to do")
		return nil
	}

	return fmt.Errorf("either workspace repository, image, container or local-folder is required")
}

func ensureLastDevContainerJson(workspaceInfo *provider2.AgentWorkspaceInfo) error {
	filePath := filepath.Join(workspaceInfo.ContentFolder, filepath.FromSlash(workspaceInfo.LastDevContainerConfig.Path))
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(filePath), 0o755)
		if err != nil {
			return fmt.Errorf("create %s: %w", filepath.Dir(filePath), err)
		}

		raw, err := json.Marshal(workspaceInfo.LastDevContainerConfig.Config)
		if err != nil {
			return fmt.Errorf("marshal devcontainer.json: %w", err)
		}

		err = os.WriteFile(filePath, raw, 0o600)
		if err != nil {
			return fmt.Errorf("write %s: %w", filePath, err)
		}
	} else if err != nil {
		return fmt.Errorf("error stating %s: %w", filePath, err)
	}

	return nil
}

func configureCredentials(ctx context.Context, cancel context.CancelFunc, workspaceInfo *provider2.AgentWorkspaceInfo, client tunnel.TunnelClient, log log.Logger) (string, string, error) {
	if workspaceInfo.Agent.InjectDockerCredentials != "true" && workspaceInfo.Agent.InjectGitCredentials != "true" {
		return "", "", nil
	}

	serverPort, err := credentials.StartCredentialsServer(ctx, cancel, client, log)
	if err != nil {
		return "", "", err
	}

	binaryPath, err := os.Executable()
	if err != nil {
		return "", "", err
	}

	if workspaceInfo.Origin == "" {
		return "", "", fmt.Errorf("workspace folder is not set")
	}

	dockerCredentials := ""
	if workspaceInfo.Agent.InjectDockerCredentials == "true" {
		dockerCredentials, err = dockercredentials.ConfigureCredentialsMachine(workspaceInfo.Origin, serverPort, log)
		if err != nil {
			return "", "", err
		}
	}

	gitCredentials := ""
	if workspaceInfo.Agent.InjectGitCredentials == "true" {
		gitCredentials = fmt.Sprintf("!'%s' agent git-credentials --port %d", binaryPath, serverPort)
		_ = os.Setenv("DEVPOD_GIT_HELPER_PORT", strconv.Itoa(serverPort))
	}

	return dockerCredentials, gitCredentials, nil
}

func installDaemon(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	if len(workspaceInfo.Agent.Exec.Shutdown) == 0 {
		return nil
	}

	log.Debugf("Installing DevPod daemon into server...")
	err := daemon.InstallDaemon(workspaceInfo.Agent.DataPath, workspaceInfo.CLIOptions.DaemonInterval, log)
	if err != nil {
		return errors.Wrap(err, "install daemon")
	}

	return nil
}

func downloadLocalFolder(ctx context.Context, workspaceDir string, client tunnel.TunnelClient, log log.Logger) error {
	log.Infof("Upload folder to server")
	stream, err := client.StreamWorkspace(ctx, &tunnel.Empty{})
	if err != nil {
		return errors.Wrap(err, "read workspace")
	}

	err = extract.Extract(tunnelserver.NewStreamReader(stream, log), workspaceDir)
	if err != nil {
		return errors.Wrap(err, "extract local folder")
	}

	return nil
}

func prepareImage(workspaceDir, image string) error {
	// create a .devcontainer.json with the image
	err := os.WriteFile(filepath.Join(workspaceDir, ".devcontainer.json"), []byte(`{
  "image": "`+image+`"
}`), 0o600)
	if err != nil {
		return err
	}

	return nil
}

func cloneRepository(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, helper string, log log.Logger) error {
	s := workspaceInfo.Workspace.Source
	log.Info("Clone repository")
	log.Infof("URL: %s\n", s.GitRepository)
	if s.GitBranch != "" {
		log.Infof("Branch: %s\n", s.GitBranch)
	}
	if s.GitCommit != "" {
		log.Infof("Commit: %s\n", s.GitCommit)
	}
	if s.GitSubPath != "" {
		log.Infof("Subpath: %s\n", s.GitSubPath)
	}
	if s.GitPRReference != "" {
		log.Infof("PR: %s\n", s.GitPRReference)
	}

	workspaceDir := workspaceInfo.ContentFolder
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
		local, _ := workspaceInfo.Agent.Local.Bool()
		if local {
			return fmt.Errorf("seems like git isn't installed on your system. Please make sure to install git and make it available in the PATH")
		}
		if err := git.InstallBinary(log); err != nil {
			return err
		}
	}

	// setup private ssh key if passed in
	extraEnv := []string{}
	if workspaceInfo.CLIOptions.SSHKey != "" {
		sshExtraEnv, err := setupSSHKey(workspaceInfo.CLIOptions.SSHKey, workspaceInfo.Agent.Path)
		if err != nil {
			return err
		}
		extraEnv = append(extraEnv, sshExtraEnv...)
	}

	// run git command
	cloner := git.NewCloner(workspaceInfo.CLIOptions.GitCloneStrategy)
	gitInfo := git.NewGitInfo(s.GitRepository, s.GitBranch, s.GitCommit, s.GitPRReference, s.GitSubPath)
	err := git.CloneRepositoryWithEnv(ctx, gitInfo, extraEnv, workspaceDir, helper, cloner, log)
	if err != nil {
		return fmt.Errorf("clone repository: %w", err)
	}

	log.Done("Successfully cloned repository")

	return nil
}

func setupSSHKey(key string, agentPath string) ([]string, error) {
	keyFile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(keyFile.Name())
	defer keyFile.Close()

	if err := writeSSHKey(keyFile, key); err != nil {
		return nil, err
	}

	if err := os.Chmod(keyFile.Name(), 0o400); err != nil {
		return nil, err
	}

	env := []string{"GIT_TERMINAL_PROMPT=0"}
	gitSSHCmd := []string{agentPath, "helper", "ssh-git-clone", "--key-file=" + keyFile.Name()}
	env = append(env, "GIT_SSH_COMMAND="+command.Quote(gitSSHCmd))

	return env, nil
}

func writeSSHKey(key *os.File, sshKey string) error {
	data, err := base64.StdEncoding.DecodeString(sshKey)
	if err != nil {
		return err
	}

	_, err = key.WriteString(string(data))
	return err
}

func installDocker(log log.Logger) error {
	if !command.Exists("docker") {
		writer := log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

		log.Debug("Installing Docker...")

		shellCommand := exec.Command("sh", "-c", scripts.InstallDocker)
		shellCommand.Stdout = writer
		shellCommand.Stderr = writer
		err := shellCommand.Run()
		if err != nil {
			return err
		}
	}

	return nil
}
