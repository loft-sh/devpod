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
	"github.com/pkg/errors"
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
	targetDir := filepath.Join(os.Getenv("HOME"), "dotfiles")

	_, err := os.Stat(targetDir)
	if err != nil {
		logger.Infof("Cloning dotfiles %s", cmd.Repository)

		gitInfo := git.NormalizeRepositoryGitInfo(cmd.Repository)

		if err := git.CloneRepository(ctx, gitInfo, targetDir, "", nil, logger); err != nil {
			return err
		}
	} else {
		logger.Info("dotfiles already set up, skipping cloning")
	}

	logger.Debugf("Entering dotfiles directory")

	err = os.Chdir(targetDir)
	if err != nil {
		return err
	}

	if cmd.InstallScript != "" {
		logger.Infof("Executing install script %s", cmd.InstallScript)
		command := "./" + strings.TrimPrefix(cmd.InstallScript, "./")

		err := ensureExecutable(command)
		if err != nil {
			return errors.Wrapf(err, "failed to make install script %s executable", command)
		}

		scriptCmd := exec.Command(command)
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
		writer := logger.Writer(logrus.InfoLevel, false)

		err := ensureExecutable(command)
		if err != nil {
			logger.Infof("Failed to make install script %s executable: %v", command, err)
			logger.Debug("Trying next location")
			continue
		}

		scriptCmd := exec.Command(command)
		scriptCmd.Stdout = writer
		scriptCmd.Stderr = writer
		err = scriptCmd.Run()
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

			// remove existing symlink and relink
			if _, err := os.Lstat(filepath.Join(os.Getenv("HOME"), file.Name())); err == nil {
				os.Remove(filepath.Join(os.Getenv("HOME"), file.Name()))
			}
			err = os.Symlink(filepath.Join(pwd, file.Name()), filepath.Join(os.Getenv("HOME"), file.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureExecutable(path string) error {
	checkCmd := exec.Command("test", "-f", path)
	err := checkCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "install script %s not found", path)
	}

	chmodCmd := exec.Command("chmod", "+x", path)
	err = chmodCmd.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to make install script %s executable", path)
	}

	return nil
}
