package workspace

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// SetupGPGCmd holds the setupGPG cmd flags
type SetupGPGCmd struct {
	*flags.GlobalFlags

	PublicKey  string
	OwnerTrust string
	SocketPath string
}

// NewSetupGPGCmd creates a new command
func NewSetupGPGCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &SetupGPGCmd{
		GlobalFlags: flags,
	}
	setupGPGCmd := &cobra.Command{
		Use:   "setup-gpg",
		Short: "setups gpg-agent forwarding in the container",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	setupGPGCmd.Flags().StringVar(&cmd.PublicKey, "publickey", "", "GPG Public keys to import in armor form")
	setupGPGCmd.Flags().StringVar(&cmd.OwnerTrust, "ownertrust", "", "GPG Owner trust to import in armor form")
	setupGPGCmd.Flags().StringVar(&cmd.SocketPath, "socketpath", "", "patht to the gpg socket forwarded")
	return setupGPGCmd
}

// will forward a local gpg-agent into the remote container
// this works by
//
// - stopping remote gpg-agent and removing the sockets
// - exporting local public keys and owner trust
// - importing those into the container
// - ensuring the gpg-agent is stopped in the container
// - starting a reverse-tunnel of the local unix socket to remote
// - ensuring paths and permissions are correctly set in the remote
func (cmd *SetupGPGCmd) Run(ctx context.Context) error {
	logger := log.Default.ErrorStreamOnly()

	logger.Debugf("Initializing gpg-agent forwarding")

	logger.Debugf("Stopping container gpg-agent")

	err := exec.Command("gpgconf", []string{"--kill", "gpg-agent"}...).Run()
	if err != nil {
		return err
	}

	publicKey, err := base64.StdEncoding.DecodeString(cmd.PublicKey)
	if err != nil {
		return err
	}

	ownerTrust, err := base64.StdEncoding.DecodeString(cmd.OwnerTrust)
	if err != nil {
		return err
	}

	logger.Debugf("gpg: importing gpg public key in container")
	gpgImportCmd := exec.Command("gpg", "--import")

	stdin, err := gpgImportCmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		_, _ = stdin.Write(publicKey)
	}()

	err = gpgImportCmd.Run()
	if err != nil {
		return err
	}

	logger.Debugf("gpg: importing gpg owner trust in container")
	gpgOwnerTrustCmd := exec.Command("gpg", "--import-ownertrust")

	stdin, err = gpgOwnerTrustCmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		_, _ = stdin.Write(ownerTrust)
	}()

	err = gpgOwnerTrustCmd.Run()
	if err != nil {
		return err
	}

	logger.Debugf("Fix paths permissions")
	out, err := exec.Command("sudo",
		"chown",
		"-R",
		strconv.Itoa(os.Getuid())+":"+strconv.Itoa(os.Getgid()),
		"/run/user",
		filepath.Dir(cmd.SocketPath),
		cmd.SocketPath,
	).CombinedOutput()

	logger.Debug(string(out))
	if err != nil {
		return err
	}

	// Now we again kill the agent and remove the socket to really be sure every
	// thing is clean

	logger.Debugf("Ensure stopping container gpg-agent")
	err = exec.Command("gpgconf", []string{"--kill", "gpg-agent"}...).Run()
	if err != nil {
		return err
	}

	logger.Debugf("Create gnupg dirs")
	err = os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".gnupg"), 0o700)
	if err != nil {
		return err
	}

	symlinks := []string{
		filepath.Join(os.Getenv("HOME"), ".gnupg", "S.gpg-agent"),
		"/run/user/" + strconv.Itoa(os.Getuid()) + "/gnupg/S.gpg-agent",
		strings.ReplaceAll(cmd.SocketPath, ".extra", ""),
	}

	for _, link := range symlinks {
		logger.Debugf("Linking extra agent to %s", link)
		_ = os.Remove(link)

		err = os.Symlink(cmd.SocketPath, link)
		if err != nil {
			return err
		}
	}

	logger.Debugf("Ensuring gpg.conf exists")
	_, err = os.Stat(filepath.Join(os.Getenv("HOME"), ".gnupg", "gpg.conf"))
	if err != nil {
		_, err = os.Create(filepath.Join(os.Getenv("HOME"), ".gnupg", "gpg.conf"))
		if err != nil {
			return err
		}
	}

	logger.Debugf("Reading gpg.conf")
	gpgConfig, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".gnupg", "gpg.conf"))
	if err != nil {
		return err
	}

	logger.Debugf("Ensuring gpg.conf has 'use-agent' property")
	if !strings.Contains(string(gpgConfig), "use-agent") {
		f, err := os.OpenFile(filepath.Join(os.Getenv("HOME"), ".gnupg", "gpg.conf"),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.WriteString("use-agent\n"); err != nil {
			return err
		}
	}

	select {}
	return nil
}
