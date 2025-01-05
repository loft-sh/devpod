package command

import (
	"os"
	"os/user"
)

func GetHome(userName string) (string, error) {
	if userName == "" {
		return os.UserHomeDir()
	}

	u, err := user.Lookup(userName)
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}
