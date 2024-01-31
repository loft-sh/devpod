package container

import (
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DaemonCmd holds the cmd flags
type DaemonCmd struct {
	Timeout string
}

// NewDaemonCmd creates a new command
func NewDaemonCmd() *cobra.Command {
	cmd := &DaemonCmd{}
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Terminates the container if timeout is reached",
		Args:  cobra.NoArgs,
		RunE:  cmd.Run,
	}
	daemonCmd.Flags().StringVar(&cmd.Timeout, "timeout", "", "The timeout to stop the container after")
	return daemonCmd
}

// Run runs the command logic
func (cmd *DaemonCmd) Run(_ *cobra.Command, _ []string) error {
	cmd.Timeout = strings.TrimSpace(cmd.Timeout)
	if cmd.Timeout == "" {
		return nil
	}

	duration, err := time.ParseDuration(cmd.Timeout)
	if err != nil {
		return errors.Wrap(err, "parse duration")
	} else if duration == 0 {
		return nil
	}

	err = os.WriteFile(agent.ContainerActivityFile, nil, 0o777)
	if err != nil {
		return err
	}

	_ = os.Chmod(agent.ContainerActivityFile, 0o777)

	// query the activity file
	for {
		time.Sleep(10 * time.Second)

		stat, err := os.Stat(agent.ContainerActivityFile)
		if err != nil {
			continue
		}

		if stat.ModTime().Add(duration).After(time.Now()) {
			continue
		}

		// kill container
		return command.Kill("1")
	}
}
