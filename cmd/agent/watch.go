package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/providerimplementation"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WatchCmd holds the up cmd flags
type WatchCmd struct{}

// NewWatchCmd creates a new ssh command
func NewWatchCmd() *cobra.Command {
	cmd := &WatchCmd{}
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Watches for activity and stops the server due to inactivity",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	return watchCmd
}

func (cmd *WatchCmd) Run(ctx context.Context) error {
	logFolder, err := agent.GetAgentDaemonLogFolder()
	if err != nil {
		return err
	}

	logger := log.NewFileLogger(filepath.Join(logFolder, "agent-daemon.log"), logrus.InfoLevel)
	logger.Infof("Starting DevPod Daemon patrol...")

	// start patrolling
	patrol(logger)

	// should never reach this
	return nil
}

func patrol(log log.Logger) {
	// make sure we don't immediately resleep on startup
	initialTouch(log)

	// loop over workspace configs and check their last ModTime
	for {
		select {
		case <-time.After(time.Minute):
			baseFolders := agent.GetBaseFolders()
			for _, baseFolder := range baseFolders {
				pattern := baseFolder + "/contexts/*/workspaces/*/" + config.WorkspaceConfigFile
				matches, err := filepath.Glob(pattern)
				if err != nil {
					log.Errorf("Error globing pattern %s: %v", pattern, err)
					continue
				}

				// check when the last touch was
				for _, match := range matches {
					err = checkForInactivity(match, log)
					if err != nil {
						log.Errorf("Error checking for inactivity: %v", err)
						continue
					}
				}
			}
		}
	}
}

func initialTouch(log log.Logger) {
	baseFolders := agent.GetBaseFolders()
	for _, baseFolder := range baseFolders {
		pattern := baseFolder + "/contexts/*/workspaces/*/" + config.WorkspaceConfigFile
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Errorf("Error globing pattern %s: %v", pattern, err)
			continue
		}

		// check when the last touch was
		now := time.Now()
		for _, match := range matches {
			err := os.Chtimes(match, now, now)
			if err != nil {
				log.Errorf("Error touching workspace config %s: %v", pattern, err)
				continue
			}
		}
	}
}

func checkForInactivity(workspaceConfig string, log log.Logger) error {
	workspace, err := parseWorkspace(workspaceConfig)
	if err != nil {
		log.Errorf("Error reading %s: %v", workspaceConfig, err)
		return nil
	}

	// check if shutdown is configured
	if workspace.AgentConfig == nil || len(workspace.AgentConfig.Exec.Shutdown) == 0 {
		return nil
	}

	// check timeout
	timeout := agent.DefaultInactivityTimeout
	if workspace.AgentConfig.Timeout != "" {
		timeout, err = time.ParseDuration(workspace.AgentConfig.Timeout)
		if err != nil {
			log.Errorf("Error parsing inactivity timeout: %v", err)
			timeout = agent.DefaultInactivityTimeout
		}
	}

	// check last access time
	stat, err := os.Stat(workspaceConfig)
	if err != nil {
		return err
	}

	// check if timeout
	now := time.Now()
	if stat.ModTime().Add(timeout).After(now) {
		return nil
	}

	// we run the timeout command now
	buf := &bytes.Buffer{}
	log.Infof("Run shutdown command for workspace %s: %s", workspaceConfig, strings.Join(workspace.AgentConfig.Exec.Shutdown, " "))
	err = providerimplementation.RunCommand(context.Background(), workspace.AgentConfig.Exec.Shutdown, workspace.Workspace, nil, buf, buf, nil)
	if err != nil {
		log.Errorf("Error running %s: %s%v", strings.Join(workspace.AgentConfig.Exec.Shutdown, " "), buf.String(), err)
		return err
	}

	log.Infof("Successful ran command: %s", buf.String())
	return nil
}

func parseWorkspace(path string) (*agent.AgentWorkspaceInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	workspace := &agent.AgentWorkspaceInfo{}
	err = json.Unmarshal(content, workspace)
	if err != nil {
		return nil, err
	}

	return workspace, nil
}
