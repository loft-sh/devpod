package helper

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/spf13/cobra"
)

// FleetServerCmd holds the fleet server cmd flags
type FleetServerCmd struct {
	*flags.GlobalFlags

	WorkspaceID string
}

// NewFleetServerCmd creates a new fleet command
func NewFleetServerCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &FleetServerCmd{
		GlobalFlags: flags,
	}
	fleetCmd := &cobra.Command{
		Use:   "fleet-server",
		Short: "Monitor fleet server activity",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}

	fleetCmd.Flags().StringVar(&cmd.WorkspaceID, "workspaceid", "", "Fleet WorkspaceID to monitor")
	return fleetCmd
}

// Run runs the command logic
func (c *FleetServerCmd) Run(cmd *cobra.Command, _ []string) error {
	logFile := filepath.Join(os.Getenv("HOME"), ".cache/JetBrains/Fleet/log/fleet.log")
	firstConnection := regexp.MustCompile(`.*Received authorization request.*`)
	connStatus := regexp.MustCompile(`.*Notify.*`)

	for {
		select {
		case <-time.After(time.Second * 10):

			log, err := os.ReadFile(logFile)
			if err != nil {
				continue
			}

			// check if we had at least one fleet client connection, before
			// this point, we don't check for connected/disconnected strings
			initialized := firstConnection.FindStringSubmatch(string(log))
			if len(initialized) == 0 {
				continue
			}

			connString := connStatus.FindAllStringSubmatch(string(log), -1)

			// if ouf last occurrence of notify if "Notify ID connected"
			// we have an active session, so let's keep alive
			if strings.Contains(connString[len(connString)-1][0], "is connected") {
				file, _ := os.Create(agent.ContainerActivityFile)
				file.Close()
			}
		case <-cmd.Context().Done():
			//context is done - either canceled or time is up for timeout
			return nil
		}
	}
}
