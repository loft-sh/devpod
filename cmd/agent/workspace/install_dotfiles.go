package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// InstallDotfilesCmd holds the installDotfiles cmd flags
type InstallDotfilesCmd struct {
	*flags.GlobalFlags

	Repository    string
	InstallScript string
}

// NewInstallDotfilesCmd creates a new command
func NewInstallDotfilesCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &InstallDotfilesCmd{
		GlobalFlags: flags,
	}
	installDotfilesCmd := &cobra.Command{
		Use:   "install-dotfiles",
		Short: "installs input dotfiles in the container",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	installDotfilesCmd.Flags().StringVar(&cmd.Repository, "repository", "", "The dotfiles repository")
	installDotfilesCmd.Flags().StringVar(&cmd.InstallScript, "install-script", "", "The dotfiles install command to execute")
	return installDotfilesCmd
}

// Run runs the command logic
func (cmd *InstallDotfilesCmd) Run(ctx context.Context) error {
	logger := log.Default.ErrorStreamOnly()

	_, err := os.Stat("dotfiles")
	if err == nil {
		logger.Info("dotfiles already set up, skipping")

		return nil
	}

	cloneArgs := []string{"clone", cmd.Repository, "dotfiles"}

	logger.Infof("Cloning dotfiles %s", cmd.Repository)

	err = git.CommandContext(ctx, cloneArgs...).Run()
	if err != nil {
		return err
	}

	logger.Debugf("Entering dotfiles directory")

	err = os.Chdir("dotfiles")
	if err != nil {
		return err
	}

	if cmd.InstallScript != "" {
		logger.Infof("Executing install script %s", cmd.InstallScript)

		scriptCmd := exec.Command("./" + cmd.InstallScript)

		writer := logger.Writer(logrus.InfoLevel, false)

		scriptCmd.Stdout = writer
		scriptCmd.Stderr = writer

		return scriptCmd.Run()
	}

	logger.Debugf("Install script not specified, trying known locations")

	return setupDotfiles(logger)
}

var scriptLocations = []string{
	"./install.sh",
	"./install",
	"./bootstrap.sh",
	"./bootstrap",
	"./script/bootstrap",
	"./setup.sh",
	"./setup",
	"./setup/setup",
}

func setupDotfiles(logger log.Logger) error {
	for _, command := range scriptLocations {
		logger.Debugf("Trying executing %s", command)

		scriptCmd := exec.Command(command)

		writer := logger.Writer(logrus.InfoLevel, false)

		scriptCmd.Stdout = writer
		scriptCmd.Stderr = writer
		err := scriptCmd.Run()
		if err != nil {
			logger.Infof("Execution of %s was unsuccessful: %v", command, err)
			logger.Debug("Trying next location")

			continue
		}

		// we successfully executed one of the commands, let's exit
		return nil
	}

	logger.Info("Finished script locations, trying to link the files")

	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// link dotfiles in directory to home
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") && !file.IsDir() {
			logger.Debugf("linking %s in home", file.Name())

			err = os.Symlink(filepath.Join(pwd, file.Name()), filepath.Join(os.Getenv("HOME"), file.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
