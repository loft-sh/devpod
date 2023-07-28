package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

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
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/port"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/random"
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

	// make sure daemon does shut us down while we are doing things
	agent.CreateWorkspaceBusyFile(workspaceInfo.Origin)
	defer agent.DeleteWorkspaceBusyFile(workspaceInfo.Origin)

	// initialize the workspace
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	tunnelClient, logger, credentialsDir, err := initWorkspace(cancelCtx, cancel, workspaceInfo, cmd.Debug, !workspaceInfo.CLIOptions.Proxy && !workspaceInfo.CLIOptions.DisableDaemon)
	if err != nil {
		err1 := clientimplementation.DeleteWorkspaceFolder(workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID, logger)
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

func initWorkspace(ctx context.Context, cancel context.CancelFunc, workspaceInfo *provider2.AgentWorkspaceInfo, debug, shouldInstallDaemon bool) (tunnel.TunnelClient, log.Logger, string, error) {
	// create a grpc client
	tunnelClient, err := tunnelserver.NewTunnelClient(os.Stdin, os.Stdout, true)
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
		if workspaceInfo.Agent.Driver == provider2.KubernetesDriver || workspaceInfo.Agent.Docker.Install == "false" {
			errChan <- nil
		} else {
			errChan <- InstallDocker(logger)
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

func prepareWorkspace(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, client tunnel.TunnelClient, helper string, log log.Logger) error {
	// check if workspace content folder exists
	_, err := os.Stat(workspaceInfo.ContentFolder)
	if err == nil {
		log.Debugf("Workspace Folder already exists")
		return nil
	}

	// make content dir
	err = os.MkdirAll(workspaceInfo.ContentFolder, 0777)
	if err != nil {
		return errors.Wrap(err, "make workspace folder")
	}

	// download provider
	binariesDir, err := agent.GetAgentBinariesDir(workspaceInfo.Agent.DataPath, workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID)
	if err != nil {
		_ = os.RemoveAll(workspaceInfo.ContentFolder)
		return fmt.Errorf("error getting workspace %s binaries dir: %w", workspaceInfo.Workspace.ID, err)
	}

	// download binaries
	_, err = binaries.DownloadBinaries(workspaceInfo.Agent.Binaries, binariesDir, log)
	if err != nil {
		_ = os.RemoveAll(workspaceInfo.ContentFolder)
		return fmt.Errorf("error downloading workspace %s binaries: %w", workspaceInfo.Workspace.ID, err)
	}

	// check what type of workspace this is
	if workspaceInfo.Workspace.Source.GitRepository != "" {
		log.Debugf("Clone Repository")
		err = CloneRepository(ctx, workspaceInfo.Agent.Local == "true", workspaceInfo.ContentFolder, workspaceInfo.Workspace.Source.GitRepository, workspaceInfo.Workspace.Source.GitBranch, workspaceInfo.Workspace.Source.GitCommit, helper, log)
		if err != nil {
			// fallback
			log.Errorf("Cloning failed: %v. Trying cloning on local machine and uploading folder", err)
			return RemoteCloneAndDownload(ctx, workspaceInfo.ContentFolder, client, log)
		}

		return nil
	} else if workspaceInfo.Workspace.Source.LocalFolder != "" {
		log.Debugf("Download Local Folder")
		return DownloadLocalFolder(ctx, workspaceInfo.ContentFolder, client, log)
	} else if workspaceInfo.Workspace.Source.Image != "" {
		log.Debugf("Prepare Image")
		return PrepareImage(workspaceInfo.ContentFolder, workspaceInfo.Workspace.Source.Image)
	}

	return fmt.Errorf("either workspace repository, image or local-folder is required")
}

func configureCredentials(ctx context.Context, cancel context.CancelFunc, workspaceInfo *provider2.AgentWorkspaceInfo, client tunnel.TunnelClient, log log.Logger) (string, string, error) {
	if workspaceInfo.Agent.InjectDockerCredentials != "true" && workspaceInfo.Agent.InjectGitCredentials != "true" {
		return "", "", nil
	}

	serverPort, err := startCredentialsServer(ctx, cancel, client, log)
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
		dockerCredentials, err = dockercredentials.ConfigureCredentialsMachine(workspaceInfo.Origin, serverPort)
		if err != nil {
			return "", "", err
		}
	}

	gitCredentials := ""
	if workspaceInfo.Agent.InjectGitCredentials == "true" {
		gitCredentials = fmt.Sprintf("%s agent git-credentials --port %d", binaryPath, serverPort)
		_ = os.Setenv("DEVPOD_GIT_HELPER_PORT", strconv.Itoa(serverPort))
	}

	return dockerCredentials, gitCredentials, nil
}

func startCredentialsServer(ctx context.Context, cancel context.CancelFunc, client tunnel.TunnelClient, log log.Logger) (int, error) {
	port, err := port.FindAvailablePort(random.InRange(13000, 17000))
	if err != nil {
		return 0, err
	}

	go func() {
		defer cancel()

		err := credentials.RunCredentialsServer(ctx, "", port, false, false, false, client, log)
		if err != nil {
			log.Errorf("Run git credentials server: %v", err)
		}
	}()

	// wait until credentials server is up
	maxWait := time.Second * 4
	now := time.Now()
Outer:
	for {
		err := PingURL(ctx, "http://localhost:"+strconv.Itoa(port))
		if err != nil {
			select {
			case <-ctx.Done():
				break Outer
			case <-time.After(time.Second):
			}
		} else {
			log.Debugf("Credentials server started...")
			break
		}

		if time.Since(now) > maxWait {
			log.Debugf("Credentials server didn't start in time...")
			break
		}
	}

	return port, nil
}

func PingURL(ctx context.Context, url string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := devpodhttp.GetHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
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

func RemoteCloneAndDownload(ctx context.Context, workspaceDir string, client tunnel.TunnelClient, log log.Logger) error {
	log.Infof("Cloning from host and upload folder to server")
	stream, err := client.GitCloneAndRead(ctx, &tunnel.Empty{})
	if err != nil {
		return errors.Wrap(err, "local cloning")
	}

	err = extract.Extract(tunnelserver.NewStreamReader(stream, log), workspaceDir)
	if err != nil {
		return errors.Wrap(err, "cloning local folder")
	}

	return nil
}

func DownloadLocalFolder(ctx context.Context, workspaceDir string, client tunnel.TunnelClient, log log.Logger) error {
	log.Infof("Upload folder to server")
	stream, err := client.ReadWorkspace(ctx, &tunnel.Empty{})
	if err != nil {
		return errors.Wrap(err, "read workspace")
	}

	err = extract.Extract(tunnelserver.NewStreamReader(stream, log), workspaceDir)
	if err != nil {
		return errors.Wrap(err, "extract local folder")
	}

	return nil
}

func PrepareImage(workspaceDir, image string) error {
	// create a .devcontainer.json with the image
	err := os.WriteFile(filepath.Join(workspaceDir, ".devcontainer.json"), []byte(`{
  "image": "`+image+`"
}`), 0666)
	if err != nil {
		return err
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
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CloneRepository(ctx context.Context, local bool, workspaceDir, repository, branch, commit, helper string, log log.Logger) error {
	// remove the credential helper or otherwise we will receive strange errors within the container
	defer func() {
		if helper != "" {
			err := gitcredentials.RemoveHelperFromPath(filepath.Join(workspaceDir, ".git", "config"))
			if err != nil {
				log.Errorf("Remove git credential helper: %v", err)
			}
		}
	}()

	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	// check if command exists
	if !command.Exists("git") {
		if local {
			return fmt.Errorf("seems like git isn't installed on your system. Please make sure to install git and make it available in the PATH")
		}

		// try to install git via apt / apk
		if !command.Exists("apt") && !command.Exists("apk") {
			// TODO: use golang git implementation
			return fmt.Errorf("couldn't find a package manager to install git")
		}

		if command.Exists("apt") {
			log.Infof("Git command is missing, try to install git with apt...")
			cmd := exec.Command("apt", "update")
			cmd.Stdout = writer
			cmd.Stderr = writer
			err := cmd.Run()
			if err != nil {
				return errors.Wrap(err, "run apt update")
			}
			cmd = exec.Command("apt", "-y", "install", "git")
			cmd.Stdout = writer
			cmd.Stderr = writer
			err = cmd.Run()
			if err != nil {
				return errors.Wrap(err, "run apt install git -y")
			}
		} else if command.Exists("apk") {
			log.Infof("Git command is missing, try to install git with apk...")
			cmd := exec.Command("apk", "update")
			cmd.Stdout = writer
			cmd.Stderr = writer
			err := cmd.Run()
			if err != nil {
				return errors.Wrap(err, "run apk update")
			}
			cmd = exec.Command("apk", "add", "git")
			cmd.Stdout = writer
			cmd.Stderr = writer
			err = cmd.Run()
			if err != nil {
				return errors.Wrap(err, "run apk add git")
			}
		}

		// is git available now?
		if !command.Exists("git") {
			return fmt.Errorf("couldn't install git")
		}

		log.Donef("Successfully installed git")
	}

	// run git command
	args := []string{"clone"}
	if helper != "" {
		args = append(args, "--config", "credential.helper="+helper)
	}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, repository, workspaceDir)
	gitCommand := git.CommandContext(ctx, args...)
	gitCommand.Stdout = writer
	gitCommand.Stderr = writer
	err := gitCommand.Run()
	if err != nil {
		return errors.Wrap(err, "error cloning repository")
	}

	if commit != "" {
		args := []string{"reset", "--hard", commit}
		gitCommand := git.CommandContext(ctx, args...)
		gitCommand.Dir = workspaceDir
		gitCommand.Stdout = writer
		gitCommand.Stderr = writer
		err := gitCommand.Run()
		if err != nil {
			return errors.Wrap(err, "error resetting head to commit")
		}
	}

	return nil
}

func InstallDocker(log log.Logger) error {
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

func CreateRunner(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (*devcontainer.Runner, error) {
	return devcontainer.NewRunner(agent.ContainerDevPodHelperLocation, agent.DefaultAgentDownloadURL(), workspaceInfo, log)
}
