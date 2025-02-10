package workspace

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/credentials"
	"github.com/loft-sh/devpod/pkg/gitcredentials"
	"github.com/loft-sh/devpod/pkg/gpg"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// SetupGPGCmd holds the setupGPG cmd flags
type SetupGPGCmd struct {
	*flags.GlobalFlags

	OwnerTrust string
	SocketPath string
	GitKey     string
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
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context(), log.Default.ErrorStreamOnly())
		},
	}
	setupGPGCmd.Flags().StringVar(&cmd.OwnerTrust, "ownertrust", "", "GPG Owner trust to import in armor form")
	setupGPGCmd.Flags().StringVar(&cmd.SocketPath, "socketpath", "", "path to the gpg socket forwarded")
	setupGPGCmd.Flags().StringVar(&cmd.GitKey, "gitkey", "", "gpg key to use for git commit signing")
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
func (cmd *SetupGPGCmd) Run(ctx context.Context, log log.Logger) error {
	log.Debugf("Initializing gpg-agent forwarding")

	log.Debugf("Fetching public key")
	rawPublicKeys, err := getPublicKeys(log)
	if err != nil {
		log.Errorf("Fetch public key: %v", err)
		return err
	}

	log.Debugf("Decoding public key")
	publicKey, err := base64.StdEncoding.DecodeString(rawPublicKeys)
	if err != nil {
		return err
	}

	log.Debugf("Decoding input owner trust")
	ownerTrust, err := base64.StdEncoding.DecodeString(cmd.OwnerTrust)
	if err != nil {
		return err
	}

	gpgConf := gpg.GPGConf{
		PublicKey:  publicKey,
		OwnerTrust: ownerTrust,
		SocketPath: cmd.SocketPath,
		GitKey:     cmd.GitKey,
	}

	log.Debugf("Stopping container gpg-agent")
	err = gpgConf.StopGpgAgent()
	if err != nil {
		log.Errorf("stop container gpg-agent: %v", err)
		return err
	}

	log.Debugf("Importing gpg public key in container")
	err = gpgConf.ImportGpgKey()
	if err != nil {
		log.Errorf("Import gpg public key in container: %v", err)
		return err
	}

	log.Debugf("Importing gpg owner trust in container")
	err = gpgConf.ImportOwnerTrust()
	if err != nil {
		log.Errorf("Import gpg owner trust in container: %v", err)
		return err
	}

	log.Debugf("Ensuring paths existence and permissions")
	err = gpgConf.SetupRemoteSocketDirTree()
	if err != nil {
		log.Errorf("Ensure paths existence and permissions: %v", err)
		return err
	}

	// Now we again kill the agent and remove the socket to really be sure every
	// thing is clean
	log.Debugf("Ensure stopping container gpg-agent")
	err = gpgConf.StopGpgAgent()
	if err != nil {
		log.Errorf("Ensure stopping container gpg-agent: %v", err)
		return err
	}

	log.Debugf("Setup local gnupg socket links")
	err = gpgConf.SetupRemoteSocketLink()
	if err != nil {
		log.Errorf("Setup local gnupg socket links: %v", err)
		return err
	}

	log.Debugf("Setup gpg.conf")
	err = gpgConf.SetupGpgConf()
	if err != nil {
		log.Errorf("Setup gpg.conf: %v", err)
		return err
	}

	if gpgConf.GitKey != "" {
		log.Debugf("Setup git signing key")
		err = gitcredentials.SetupGpgGitKey(gpgConf.GitKey)
		if err != nil {
			log.Errorf("Setup git signing key: %v", err)
			return err
		}
	}

	return nil
}

func getPublicKeys(log log.Logger) (string, error) {
	port, err := credentials.GetPort()
	if err != nil {
		return "", fmt.Errorf("get port: %w", err)
	}

	out, err := credentials.PostWithRetry(port, "gpg-public-keys", nil, log)
	if err != nil {
		return "", fmt.Errorf("get public gpg keys: %w", err)
	}

	return string(out), nil
}
