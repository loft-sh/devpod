package gpg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/loft-sh/log"

	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type GPGConf struct {
	PublicKey  []byte
	OwnerTrust []byte
	SocketPath string
	GitKey     string
}

func IsGpgTunnelRunning(
	user string,
	ctx context.Context,
	client *ssh.Client,
	log log.Logger,
) bool {
	writer := log.ErrorStreamOnly().Writer(logrus.InfoLevel, false)
	defer writer.Close()

	command := "gpg -K"
	if user != "" && user != "root" {
		command = fmt.Sprintf("su -c \"%s\" '%s'", command, user)
	}

	// capture the output, if it's empty it means we don't have gpg-forwarding
	var out bytes.Buffer
	err := devssh.Run(ctx, client, command, nil, &out, writer, nil)

	return err == nil && len(out.Bytes()) > 1
}

func GetHostPubKey() ([]byte, error) {
	return exec.Command("gpg", "--armor", "--export").Output()
}

func GetHostOwnerTrust() ([]byte, error) {
	return exec.Command("gpg", "--export-ownertrust").Output()
}

func (g *GPGConf) StopGpgAgent() error {
	return exec.Command("gpgconf", []string{"--kill", "gpg-agent"}...).Run()
}

func (g *GPGConf) ImportGpgKey() error {
	gpgImportCmd := exec.Command("gpg", "--import")

	stdin, err := gpgImportCmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		_, _ = stdin.Write(g.PublicKey)
	}()

	out, err := gpgImportCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("import gpg public key: %s %w", out, err)
	}

	return nil
}

func (g *GPGConf) ImportOwnerTrust() error {
	gpgOwnerTrustCmd := exec.Command("gpg", "--import-ownertrust")

	stdin, err := gpgOwnerTrustCmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		_, _ = stdin.Write(g.OwnerTrust)
	}()

	return gpgOwnerTrustCmd.Run()
}

func (g *GPGConf) SetupGpgConf() error {
	_, err := os.Stat(g.getConfigPath())
	if err != nil {
		_, err = os.Create(g.getConfigPath())
		if err != nil {
			return err
		}
	}

	gpgConfig, err := os.ReadFile(g.getConfigPath())
	if err != nil {
		return err
	}

	if !strings.Contains(string(gpgConfig), "use-agent") {
		f, err := os.OpenFile(g.getConfigPath(),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.WriteString("use-agent\n"); err != nil {
			return err
		}
	}

	return nil
}

func (g *GPGConf) SetupRemoteSocketDirTree() error {
	err := exec.Command("sudo", "mkdir", "-p", "/run/user", filepath.Dir(g.SocketPath)).Run()
	if err != nil {
		return err
	}

	return exec.Command("sudo",
		"chown",
		"-R",
		strconv.Itoa(os.Getuid())+":"+strconv.Itoa(os.Getgid()),
		"/run/user",
		filepath.Dir(g.SocketPath),
		g.SocketPath,
	).Run()
}

// This function will normalize the location of the forwarded socket.
// the forwarding that happens in pkg/ssh/forward.go will forward the socket in
// the same path (eg. /Users/foo/.gnupg/S.gpg-agent)
// This function will use hardlinks to normalize it to where linux usually
// expects the socket to be.
func (g *GPGConf) SetupRemoteSocketLink() error {
	err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".gnupg"), 0o700)
	if err != nil {
		return err
	}

	err = exec.Command("sudo", "ln", "-s", "-f", g.SocketPath, "/tmp/S.gpg-agent").Run()
	if err != nil {
		return err
	}

	symlinks := []string{
		filepath.Join(os.Getenv("HOME"), ".gnupg", "S.gpg-agent"),
		"/run/user/" + strconv.Itoa(os.Getuid()) + "/gnupg/S.gpg-agent",
	}

	for _, link := range symlinks {
		_ = os.Remove(link)
		_ = os.MkdirAll(filepath.Dir(link), 0o755)

		err = os.Symlink("/tmp/S.gpg-agent", link)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GPGConf) getConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".gnupg", "gpg.conf")
}
