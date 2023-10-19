package workspace

import (
	"context"
	"encoding/base64"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/gpg"
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

	logger.Debugf("Decoding input public key")
	publicKey, err := base64.StdEncoding.DecodeString(cmd.PublicKey)
	if err != nil {
		return err
	}

	logger.Debugf("Decoding input owner trust")
	ownerTrust, err := base64.StdEncoding.DecodeString(cmd.OwnerTrust)
	if err != nil {
		return err
	}

	gpgConf := gpg.GPGConf{
		PublicKey:  publicKey,
		OwnerTrust: ownerTrust,
		SocketPath: cmd.SocketPath,
	}

	logger.Debugf("Stopping container gpg-agent")
	err = gpgConf.StopGpgAgent()
	if err != nil {
		return err
	}

	logger.Debugf("gpg: importing gpg public key in container")
	err = gpgConf.ImportGpgKey()
	if err != nil {
		return err
	}

	logger.Debugf("gpg: importing gpg owner trust in container")
	err = gpgConf.ImportOwnerTrust()
	if err != nil {
		return err
	}

	logger.Debugf("Ensuring paths existence and permissions")
	err = gpgConf.SetupRemoteSocketDirTree()
	if err != nil {
		return err
	}

	// Now we again kill the agent and remove the socket to really be sure every
	// thing is clean
	logger.Debugf("Ensure stopping container gpg-agent")
	err = gpgConf.StopGpgAgent()
	if err != nil {
		return err
	}

	logger.Debugf("Setup local gnupg socket links")
	err = gpgConf.SetupRemoteSocketLink()
	if err != nil {
		return err
	}

	logger.Debugf("Setup gpg.conf")
	err = gpgConf.SetupGpgConf()
	if err != nil {
		return err
	}

	return nil
}
