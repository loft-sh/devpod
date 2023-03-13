package ssh

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"sync"

	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/mitchellh/go-homedir"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

var (
	DevPodSSHHostKeyFile    = "id_devpod_rsa_host"
	DevPodSSHPrivateKeyFile = "id_devpod_rsa"
	DevPodSSHPublicKeyFile  = "id_devpod_rsa.pub"
)

var keyLock sync.Mutex

func rsaKeyGen() (privateKey string, publicKey string, err error) {
	privateKeyRaw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", errors.Errorf("generate private key: %v", err)
	}

	return generateKeys(pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKeyRaw),
	}, privateKeyRaw)
}

func generateKeys(block pem.Block, cp crypto.Signer) (privateKey string, publicKey string, err error) {
	pkBytes := pem.EncodeToMemory(&block)
	privateKey = string(pkBytes)

	publicKeyRaw := cp.Public()
	p, err := ssh.NewPublicKey(publicKeyRaw)
	if err != nil {
		return "", "", err
	}
	publicKey = string(ssh.MarshalAuthorizedKey(p))

	return privateKey, publicKey, nil
}

func makeHostKey() (string, error) {
	privKey, _, err := rsaKeyGen()
	if err != nil {
		return "", err
	}

	return privKey, err
}

func makeSSHKeyPair() (string, string, error) {
	privKey, pubKey, err := rsaKeyGen()
	if err != nil {
		return "", "", err
	}

	return pubKey, privKey, err
}

func GetPrivateKeyRaw(context, workspaceID string) ([]byte, error) {
	workspaceDir, err := provider.GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return nil, err
	}

	return GetPrivateKeyRawBase(workspaceDir)
}

func GetDevPodKeysDir() string {
	dir, err := homedir.Dir()
	if err == nil {
		tempDir := filepath.Join(dir, ".devpod", "keys")
		err = os.MkdirAll(tempDir, 0755)
		if err == nil {
			return tempDir
		}
	}

	tempDir := os.TempDir()
	return filepath.Join(tempDir, "devpod-ssh")
}

func GetDevPodHostKey() (string, error) {
	tempDir := GetDevPodKeysDir()
	return GetHostKeyBase(tempDir)
}

func GetDevPodPublicKey() (string, error) {
	tempDir := GetDevPodKeysDir()
	return GetPublicKeyBase(tempDir)
}

func GetDevPodPrivateKeyRaw() ([]byte, error) {
	tempDir := GetDevPodKeysDir()
	return GetPrivateKeyRawBase(tempDir)
}

func GetHostKey(context, workspaceID string) (string, error) {
	workspaceDir, err := provider.GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return "", err
	}

	return GetHostKeyBase(workspaceDir)
}

func GetPrivateKeyRawBase(dir string) ([]byte, error) {
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

func GetHostKeyBase(dir string) (string, error) {
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

func GetPublicKeyBase(dir string) (string, error) {
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
	workspaceDir, err := provider.GetWorkspaceDir(context, workspaceID)
	if err != nil {
		return "", err
	}

	return GetPublicKeyBase(workspaceDir)
}
