//go:build !linux

package zed

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/skratchdot/open-golang/open"

	"github.com/loft-sh/devpod/pkg/config"
)

// Open first finds the zed binary for the local platform and then opens the zed editor with the given workspace folder
func Open(ctx context.Context, values map[string]config.OptionValue, userName, workspaceFolder, workspaceID string, log log.Logger) error {
	log.Info("Opening Zed editor...")

	if len(workspaceFolder) == 0 || workspaceFolder[0] != '/' {
		workspaceFolder = fmt.Sprintf("/%s", workspaceFolder)
	}

	sshHost := fmt.Sprintf("%s.devpod%s", workspaceID, workspaceFolder)
	openURL := fmt.Sprintf("zed://ssh/%s", sshHost)
	err := open.Run(openURL)
	if err != nil {
		log.Debugf("Starting Zed caused error: %v", err)
		log.Errorf("Seems like you don't have Zed installed on your computer locally")
		return err
	}

	return nil
}
