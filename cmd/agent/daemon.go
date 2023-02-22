package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/log"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/provider/providerimplementation"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DaemonCmd holds the cmd flags
type DaemonCmd struct{}

// NewDaemonCmd creates a new command
func NewDaemonCmd() *cobra.Command {
	cmd := &DaemonCmd{}
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Watches for activity and stops the server due to inactivity",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run(context.Background())
		},
	}
	return daemonCmd
}

func (cmd *DaemonCmd) Run(ctx context.Context) error {
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
			doOnce(log)
		}
	}
}

func doOnce(log log.Logger) {
	var latestActivity *time.Time
	var workspace *provider2.AgentWorkspaceInfo

	baseFolders := agent.GetBaseFolders()
	for _, baseFolder := range baseFolders {
		pattern := baseFolder + "/contexts/*/workspaces/*/" + provider2.WorkspaceConfigFile
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Errorf("Error globing pattern %s: %v", pattern, err)
			continue
		}

		// check when the last touch was
		for _, match := range matches {
			activity, activityWorkspace, err := getActivity(match, log)
			if err != nil {
				log.Errorf("Error checking for inactivity: %v", err)
				continue
			} else if activity == nil {
				continue
			}

			if latestActivity == nil || activity.After(*latestActivity) {
				latestActivity = activity
				workspace = activityWorkspace
			}
		}
	}

	// should we run shutdown command?
	if latestActivity == nil {
		return
	}

	// check timeout
	timeout := agent.DefaultInactivityTimeout
	if workspace.Workspace.Provider.Agent.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(workspace.Workspace.Provider.Agent.Timeout)
		if err != nil {
			log.Errorf("Error parsing inactivity timeout: %v", err)
			timeout = agent.DefaultInactivityTimeout
		}
	}
	if latestActivity.Add(timeout).After(time.Now()) {
		return
	}

	// we run the timeout command now
	buf := &bytes.Buffer{}
	log.Infof("Run shutdown command for workspace %s: %s", workspace.Workspace.ID, strings.Join(workspace.Workspace.Provider.Agent.Exec.Shutdown, " "))
	err := providerimplementation.RunCommand(context.Background(), workspace.Workspace.Provider.Agent.Exec.Shutdown, &workspace.Workspace, nil, buf, buf, nil)
	if err != nil {
		log.Errorf("Error running %s: %s%v", strings.Join(workspace.Workspace.Provider.Agent.Exec.Shutdown, " "), buf.String(), err)
		return
	}

	log.Infof("Successful ran command: %s", buf.String())
}

func initialTouch(log log.Logger) {
	baseFolders := agent.GetBaseFolders()
	for _, baseFolder := range baseFolders {
		pattern := baseFolder + "/contexts/*/workspaces/*/" + provider2.WorkspaceConfigFile
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

func getActivity(workspaceConfig string, log log.Logger) (*time.Time, *provider2.AgentWorkspaceInfo, error) {
	workspace, err := parseWorkspace(workspaceConfig)
	if err != nil {
		log.Errorf("Error reading %s: %v", workspaceConfig, err)
		return nil, nil, nil
	}

	// check if shutdown is configured
	if len(workspace.Workspace.Provider.Agent.Exec.Shutdown) == 0 {
		return nil, nil, nil
	}

	// check last access time
	stat, err := os.Stat(workspaceConfig)
	if err != nil {
		return nil, nil, err
	}

	// check if timeout
	t := stat.ModTime()
	return &t, workspace, nil
}

func parseWorkspace(path string) (*provider2.AgentWorkspaceInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	workspace := &provider2.AgentWorkspaceInfo{}
	err = json.Unmarshal(content, workspace)
	if err != nil {
		return nil, err
	}

	return workspace, nil
}
