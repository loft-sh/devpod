package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/agent/tunnel"
	"github.com/loft-sh/devpod/pkg/command"
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
	workspaceInfo, err := agent.WriteWorkspaceInfo(cmd.WorkspaceInfo)
	if err != nil {
		return fmt.Errorf("error parsing workspace info: %v", err)
	}

	// check if we need to become root
	shouldExit, err := agent.RerunAsRoot(workspaceInfo)
	if err != nil {
		return fmt.Errorf("rerun as root: %v", err)
	} else if shouldExit {
		return nil
	}

	// create a grpc client
	tunnelClient, err := agent.NewTunnelClient(os.Stdin, os.Stdout, true)
	if err != nil {
		return fmt.Errorf("error creating tunnel client: %v", err)
	}

	// create debug logger
	logger := agent.NewTunnelLogger(ctx, tunnelClient, cmd.Debug)
	err = cmd.up(ctx, workspaceInfo, tunnelClient, logger)
	if err != nil {
		return errors.Wrap(err, "devcontainer up")
	}

	return nil
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
	return devcontainer.NewRunner(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, workspaceInfo, log)
}
