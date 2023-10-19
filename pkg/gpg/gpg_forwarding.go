package gpg

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type GPGConf struct {
	PublicKey  []byte
	OwnerTrust []byte
	SocketPath string
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

	return gpgImportCmd.Run()
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

func (g *GPGConf) SetupRemoteSocketDirtree() error {
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

func (g *GPGConf) SetupRemoteSocketLink() error {
	err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".gnupg"), 0o700)
	if err != nil {
		return err
	}

	err = exec.Command("sudo", "ln", "-f", g.SocketPath, "/tmp/S.gpg-agent").Run()
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
