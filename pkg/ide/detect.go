package ide

import (
	"fmt"
	"github.com/loft-sh/devpod/pkg/command"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"strings"
)

func Detect() provider2.IDE {
	if command.Exists("code") {
		return provider2.IDEVSCode
	}

	return provider2.IDEOpenVSCode
}

var allowedIDEs = []provider2.IDE{
	provider2.IDENone,
	provider2.IDEVSCode,
	provider2.IDEOpenVSCode,
	provider2.IDEGoland,
	provider2.IDEPyCharm,
	provider2.IDEPhpStorm,
	provider2.IDEIntellij,
	provider2.IDECLion,
	provider2.IDERider,
	provider2.IDERubyMine,
	provider2.IDEWebStorm,
}

func Parse(ide string) (provider2.IDE, error) {
	ide = strings.ToLower(ide)
	for _, match := range allowedIDEs {
		if string(match) == ide {
			return match, nil
		}
	}

	return "", fmt.Errorf("unrecognized ide %s, please use one of: %v", ide, allowedIDEs)
}
