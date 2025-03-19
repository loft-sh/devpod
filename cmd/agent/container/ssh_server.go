package container

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/cmd/flags"
	helperssh "github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/ssh/server/port"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const BaseLogDir = "/var/devpod"

// SSHServerCmd holds the ssh server cmd flags
type SSHServerCmd struct {
	*flags.GlobalFlags

	Address    string
	Workdir    string
	RemoteUser string
}

// NewSSHServerCmd creates a new ssh command
func NewSSHServerCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SSHServerCmd{
		GlobalFlags: flags,
	}
	sshCmd := &cobra.Command{
		Use:   "ssh-server",
		Short: "Starts the container ssh server",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	sshCmd.Flags().StringVar(&cmd.Address, "address", fmt.Sprintf("127.0.0.1:%d", helperssh.DefaultUserPort), "Address to listen to")
	sshCmd.Flags().StringVar(&cmd.RemoteUser, "remote-user", "", "The remote user for this workspace")
	sshCmd.Flags().StringVar(&cmd.Workdir, "workdir", "", "Directory where commands will run on the host")
	return sshCmd
}

// Run runs the command logic
func (cmd *SSHServerCmd) Run(_ *cobra.Command, _ []string) error {
	logger := getFileLogger(cmd.RemoteUser, cmd.Debug)
	server, err := helperssh.NewContainerServer(cmd.Address, cmd.Workdir, logger)
	if err != nil {
		return err
	}

	// check if ssh is already running at that port
	available, err := port.IsAvailable(cmd.Address)
	if !available {
		if err != nil {
			return fmt.Errorf("address %s already in use: %w", cmd.Address, err)
		}

		log.Default.ErrorStreamOnly().Info("address %s already in use", cmd.Address)
		return nil
	}

	return server.ListenAndServe()
}

func getFileLogger(remoteUser string, debug bool) log.Logger {
	logLevel := logrus.InfoLevel
	if debug {
		logLevel = logrus.DebugLevel
	}
	fallback := log.NewDiscardLogger(logLevel)

	targetFolder := filepath.Join(os.TempDir(), ".devpod")
	if remoteUser != "" {
		targetFolder = filepath.Join(BaseLogDir, remoteUser)
	}
	err := os.MkdirAll(targetFolder, 0o755)
	if err != nil {
		return fallback
	}

	return log.NewFileLogger(filepath.Join(targetFolder, "ssh.log"), logLevel)
}
