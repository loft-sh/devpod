package ssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/loft-sh/devpod/pkg/config"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

var (
	DevPodSSHHostKeyFile    = "id_devpod_host_ecdsa"
	DevPodSSHPrivateKeyFile = "id_devpod_ecdsa"
	DevPodSSHPublicKeyFile  = "id_devpod_ecdsa.pub"
)

var keyLock sync.Mutex

func generatePrivateKey() (*ecdsa.PrivateKey, string, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}

	// generate and write private key as PEM
	var privateKeyBuf strings.Builder
	b, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, "", err
	}
	privateKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	}
	if err := pem.Encode(&privateKeyBuf, privateKeyPEM); err != nil {
		return nil, "", err
	}

	return privateKey, privateKeyBuf.String(), nil
}

func makeHostKey() (string, error) {
	_, privKeyStr, err := generatePrivateKey()
	if err != nil {
		return "", err
	}
	return privKeyStr, nil
}

func makeSSHKeyPair() (string, string, error) {
	privateKey, privKeyStr, err := generatePrivateKey()
	if err != nil {
		return "", "", err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))
	return pubKeyBuf.String(), privKeyStr, nil
}

func GetPrivateKeyRaw(context, workspaceID string) ([]byte, error) {
	workspaceDir, err := config.GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return nil, err
	}

	return getPrivateKeyRawBase(workspaceDir)
}

func getTempDir() string {
	tempDir := os.TempDir()
	return filepath.Join(tempDir, "devpod-ssh")
}

func GetTempHostKey() (string, error) {
	tempDir := getTempDir()
	return getHostKeyBase(tempDir)
}

func GetTempPublicKey() (string, error) {
	tempDir := getTempDir()
	return getPublicKeyBase(tempDir)
}

func GetTempPrivateKeyRaw() ([]byte, error) {
	tempDir := getTempDir()
	return getPrivateKeyRawBase(tempDir)
}

func GetHostKey(context, workspaceID string) (string, error) {
	workspaceDir, err := config.GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return "", err
	}

	return getHostKeyBase(workspaceDir)
}

func getPrivateKeyRawBase(dir string) ([]byte, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	// check if key pair exists
	privateKeyFile := filepath.Join(dir, DevPodSSHPrivateKeyFile)
	publicKeyFile := filepath.Join(dir, DevPodSSHPublicKeyFile)
	_, err = os.Stat(privateKeyFile)
	if err != nil {
		pubKey, privateKey, err := makeSSHKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "generate key pair")
		}

		err = os.WriteFile(publicKeyFile, []byte(pubKey), 0644)
		if err != nil {
			return nil, errors.Wrap(err, "write public ssh key")
		}

		err = os.WriteFile(privateKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return nil, errors.Wrap(err, "write private ssh key")
		}
	}

	// read private key
	out, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "read private ssh key")
	}

	return out, nil
}

func getHostKeyBase(dir string) (string, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}

	// check if key pair exists
	hostKeyFile := filepath.Join(dir, DevPodSSHHostKeyFile)
	_, err = os.Stat(hostKeyFile)
	if err != nil {
		privateKey, err := makeHostKey()
		if err != nil {
			return "", errors.Wrap(err, "generate host key")
		}

		err = os.WriteFile(hostKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write host key")
		}
	}

	// read public key
	out, err := os.ReadFile(hostKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "read host ssh key")
	}

	return base64.StdEncoding.EncodeToString(out), nil
}

func getPublicKeyBase(dir string) (string, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}

	// check if key pair exists
	privateKeyFile := filepath.Join(dir, DevPodSSHPrivateKeyFile)
	publicKeyFile := filepath.Join(dir, DevPodSSHPublicKeyFile)
	_, err = os.Stat(privateKeyFile)
	if err != nil {
		pubKey, privateKey, err := makeSSHKeyPair()
		if err != nil {
			return "", errors.Wrap(err, "generate key pair")
		}

		err = os.WriteFile(publicKeyFile, []byte(pubKey), 0644)
		if err != nil {
			return "", errors.Wrap(err, "write public ssh key")
		}

		err = os.WriteFile(privateKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write private ssh key")
		}
	}

	// read public key
	out, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "read public ssh key")
	}

	return base64.StdEncoding.EncodeToString(out), nil
}

func GetPublicKey(context, workspaceID string) (string, error) {
	workspaceDir, err := config.GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return "", err
	}

	return getPublicKeyBase(workspaceDir)
}
