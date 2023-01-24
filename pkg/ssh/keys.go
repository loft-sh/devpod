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
	DevPodSSHFolder         = "ssh"
	DevPodSSHHostKeyFile    = "id_devpod_host_ecdsa"
	DevPodSSHPrivateKeyFile = "id_devpod_ecdsa"
	DevPodSSHPublicKeyFile  = "id_devpod_ecdsa.pub"
)

func init() {
	configDir, _ := config.GetConfigDir()
	DevPodSSHFolder = filepath.Join(configDir, DevPodSSHFolder)
	DevPodSSHHostKeyFile = filepath.Join(DevPodSSHFolder, DevPodSSHHostKeyFile)
	DevPodSSHPrivateKeyFile = filepath.Join(DevPodSSHFolder, DevPodSSHPrivateKeyFile)
	DevPodSSHPublicKeyFile = filepath.Join(DevPodSSHFolder, DevPodSSHPublicKeyFile)
}

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

func MakeHostKey() (string, error) {
	_, privKeyStr, err := generatePrivateKey()
	if err != nil {
		return "", err
	}
	return privKeyStr, nil
}

func MakeSSHKeyPair() (string, string, error) {
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

func GetPrivateKeyRaw() ([]byte, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	_, err := os.Stat(DevPodSSHFolder)
	if err != nil {
		err = os.MkdirAll(DevPodSSHFolder, 0755)
		if err != nil {
			return nil, err
		}
	}

	// check if key pair exists
	_, err = os.Stat(DevPodSSHPrivateKeyFile)
	if err != nil {
		pubKey, privateKey, err := MakeSSHKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "generate key pair")
		}

		err = os.WriteFile(DevPodSSHPublicKeyFile, []byte(pubKey), 0644)
		if err != nil {
			return nil, errors.Wrap(err, "write public ssh key")
		}

		err = os.WriteFile(DevPodSSHPrivateKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return nil, errors.Wrap(err, "write private ssh key")
		}
	}

	// read private key
	out, err := os.ReadFile(DevPodSSHPrivateKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "read private ssh key")
	}

	return out, nil
}

func GetHostKey() (string, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	_, err := os.Stat(DevPodSSHFolder)
	if err != nil {
		err = os.MkdirAll(DevPodSSHFolder, 0755)
		if err != nil {
			return "", err
		}
	}

	// check if key pair exists
	_, err = os.Stat(DevPodSSHHostKeyFile)
	if err != nil {
		privateKey, err := MakeHostKey()
		if err != nil {
			return "", errors.Wrap(err, "generate host key")
		}

		err = os.WriteFile(DevPodSSHHostKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write host key")
		}
	}

	// read public key
	out, err := os.ReadFile(DevPodSSHHostKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "read host ssh key")
	}

	return base64.StdEncoding.EncodeToString(out), nil
}

func GetPublicKey() (string, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	_, err := os.Stat(DevPodSSHFolder)
	if err != nil {
		err = os.MkdirAll(DevPodSSHFolder, 0755)
		if err != nil {
			return "", err
		}
	}

	// check if key pair exists
	_, err = os.Stat(DevPodSSHPrivateKeyFile)
	if err != nil {
		pubKey, privateKey, err := MakeSSHKeyPair()
		if err != nil {
			return "", errors.Wrap(err, "generate key pair")
		}

		err = os.WriteFile(DevPodSSHPublicKeyFile, []byte(pubKey), 0644)
		if err != nil {
			return "", errors.Wrap(err, "write public ssh key")
		}

		err = os.WriteFile(DevPodSSHPrivateKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write private ssh key")
		}
	}

	// read public key
	out, err := os.ReadFile(DevPodSSHPublicKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "read public ssh key")
	}

	return base64.StdEncoding.EncodeToString(out), nil
}
