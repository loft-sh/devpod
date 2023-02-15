package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/compress"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/daemon"
	"github.com/loft-sh/devpod/pkg/devcontainer"
	config2 "github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/extract"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// UpCmd holds the up cmd flags
type UpCmd struct {
	*flags.GlobalFlags

	WorkspaceInfo string
}

// NewUpCmd creates a new ssh command
func NewUpCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &UpCmd{
		GlobalFlags: flags,
	}
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Starts a new devcontainer",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			cmd.Run(context.Background())
		},
	}
	upCmd.Flags().StringVar(&cmd.WorkspaceInfo, "workspace-info", "", "The workspace info")
	_ = upCmd.MarkFlagRequired("workspace-info")
	return upCmd
}

// Run runs the command logic
func (cmd *UpCmd) Run(ctx context.Context) {
	// get workspace
	workspaceInfo, err := getWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error parsing workspace info: %v", err)
		os.Exit(1)
	}

	// check if we need to become root
	shouldExit, err := rerunAsRoot(workspaceInfo)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Rerun as root: %v", err)
		os.Exit(1)
	} else if shouldExit {
		return
	}

	// create a grpc client
	tunnelClient, err := agent.NewTunnelClient(os.Stdin, os.Stdout, true)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error creating tunnel client: %v", err)
		os.Exit(1)
	}

	// create debug logger
	logger := agent.NewTunnelLogger(ctx, tunnelClient, cmd.Debug)
	err = cmd.up(ctx, workspaceInfo, tunnelClient, logger)
	if err != nil {
		logger.Fatalf("DevPod Agent Error: %v", err)
	}
}

func (cmd *UpCmd) up(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, tunnelClient tunnel.TunnelClient, logger log.Logger) error {
	// install docker in background
	errChan := make(chan error)
	go func() {
		errChan <- InstallDocker(logger)
	}()

	// prepare workspace
	err := cmd.prepareWorkspace(ctx, workspaceInfo, tunnelClient, logger)
	if err != nil {
		return err
	}

	// install daemon
	err = installDaemon(workspaceInfo, logger)
	if err != nil {
		logger.Errorf("Install DevPod Daemon: %v", err)
	}

	// wait until docker is installed
	err = <-errChan
	if err != nil {
		return errors.Wrap(err, "install docker")
	}

	// create devcontainer
	result, err := DevContainerUp(workspaceInfo, logger)
	if err != nil {
		return err
	}

	// send result
	out, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tunnelClient.SendResult(ctx, &tunnel.Result{Message: string(out)})
	if err != nil {
		return errors.Wrap(err, "send result")
	}

	return nil
}

func (cmd *UpCmd) prepareWorkspace(ctx context.Context, workspaceInfo *provider2.AgentWorkspaceInfo, client tunnel.TunnelClient, log log.Logger) error {
	_, err := os.Stat(workspaceInfo.Folder)
	if err == nil {
		return nil
	}

	// make content dir
	err = os.MkdirAll(workspaceInfo.Folder, 0777)
	if err != nil {
		return errors.Wrap(err, "make workspace folder")
	}

	// check what type of workspace this is
	if workspaceInfo.Workspace.Source.GitRepository != "" {
		return CloneRepository(workspaceInfo.Folder, workspaceInfo.Workspace.Source.GitRepository, log)
	} else if workspaceInfo.Workspace.Source.LocalFolder != "" {
		return DownloadLocalFolder(ctx, workspaceInfo.Folder, client, log)
	} else if workspaceInfo.Workspace.Source.Image != "" {
		return PrepareImage(workspaceInfo.Folder, workspaceInfo.Workspace.Source.Image)
	}

	return fmt.Errorf("either workspace repository, image or local-folder is required")
}

func rerunAsRoot(workspaceInfo *provider2.AgentWorkspaceInfo) (bool, error) {
	// check if root is required
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		return false, nil
	}

	// check if we can reach docker with no problems
	dockerRootRequired, err := dockerReachable()
	if err != nil {
		return false, nil
	}

	// check if daemon needs to be installed
	agentRootRequired := false
	if runtime.GOOS == "linux" && len(workspaceInfo.Workspace.Provider.Agent.Exec.Shutdown) > 0 {
		agentRootRequired = true
	}

	// check if root required
	if !dockerRootRequired && !agentRootRequired {
		return false, nil
	}

	// execute ourself as root
	binary, err := os.Executable()
	if err != nil {
		return false, err
	}

	// call ourself
	args := []string{binary}
	args = append(args, os.Args[1:]...)
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return false, errors.Wrap(err, "rerun as root")
	}

	return true, nil
}

func dockerReachable() (bool, error) {
	if !command.Exists("docker") {
		// we need root to install docker
		return true, nil
	}

	_, err := exec.Command("docker", "ps").CombinedOutput()
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return true, nil
		}

		return false, errors.Wrap(err, "docker ps")
	}

	return false, nil
}

func installDaemon(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) error {
	if len(workspaceInfo.Workspace.Provider.Agent.Exec.Shutdown) == 0 {
		return nil
	}

	log.Debugf("Installing DevPod daemon into server...")
	err := daemon.InstallDaemon(log)
	if err != nil {
		return errors.Wrap(err, "install daemon")
	}

	return nil
}

func readAgentWorkspaceInfo(context, id string) (*provider2.AgentWorkspaceInfo, error) {
	// get workspace folder
	workspaceDir, err := agent.GetAgentWorkspaceDir(context, id)
	if err != nil {
		return nil, err
	}

	// read workspace config
	out, err := os.ReadFile(filepath.Join(workspaceDir, config.WorkspaceConfigFile))
	if err != nil {
		return nil, err
	}

	// json unmarshal
	workspaceInfo := &provider2.AgentWorkspaceInfo{}
	err = json.Unmarshal(out, workspaceInfo)
	if err != nil {
		return nil, errors.Wrap(err, "parse workspace info")
	}

	workspaceInfo.Folder = agent.GetAgentWorkspaceContentDir(workspaceDir)
	return workspaceInfo, nil
}

func getWorkspaceInfo(workspaceInfoRaw string) (*provider2.AgentWorkspaceInfo, error) {
	decoded, err := compress.Decompress(workspaceInfoRaw)
	if err != nil {
		return nil, errors.Wrap(err, "decode workspace info")
	}

	workspaceInfo := &provider2.AgentWorkspaceInfo{}
	err = json.Unmarshal([]byte(decoded), workspaceInfo)
	if err != nil {
		return nil, errors.Wrap(err, "parse workspace info")
	}

	// write to workspace folder
	workspaceDir, err := agent.CreateAgentWorkspaceDir(workspaceInfo.Workspace.Context, workspaceInfo.Workspace.ID)
	if err != nil {
		return nil, err
	}

	// write workspace config
	err = os.WriteFile(filepath.Join(workspaceDir, config.WorkspaceConfigFile), []byte(decoded), 0666)
	if err != nil {
		return nil, fmt.Errorf("write workspace config file")
	}

	workspaceInfo.Folder = agent.GetAgentWorkspaceContentDir(workspaceDir)
	return workspaceInfo, nil
}

func DownloadLocalFolder(ctx context.Context, workspaceDir string, client tunnel.TunnelClient, log log.Logger) error {
	log.Infof("Upload folder to server")
	stream, err := client.ReadWorkspace(ctx, &tunnel.Empty{})
	if err != nil {
		return errors.Wrap(err, "read workspace")
	}

	err = extract.Extract(agent.NewStreamReader(stream), workspaceDir)
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

func DevContainerUp(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) (*config2.Result, error) {
	result, err := createRunner(workspaceInfo, log).Up()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CloneRepository(workspaceDir, repository string, log log.Logger) error {
	// run git command
	writer := log.Writer(logrus.InfoLevel, false)
	defer writer.Close()

	gitCommand := exec.Command("git", "clone", repository, workspaceDir)
	gitCommand.Stdout = writer
	gitCommand.Stderr = writer
	err := gitCommand.Run()
	if err != nil {
		return errors.Wrap(err, "error cloning repository")
	}

	return nil
}

func InstallDocker(log log.Logger) error {
	if !command.Exists("docker") {
		writer := log.Writer(logrus.InfoLevel, false)
		defer writer.Close()

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

func createRunner(workspaceInfo *provider2.AgentWorkspaceInfo, log log.Logger) *devcontainer.Runner {
	return devcontainer.NewRunner(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, workspaceInfo.Folder, workspaceInfo.Workspace.ID, log)
}
