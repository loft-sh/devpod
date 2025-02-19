package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform"
	platformdaemon "github.com/loft-sh/devpod/pkg/platform/daemon"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// StatusCmd holds the cmd flags
type StatusCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewStatusCmd creates a new command
func NewStatusCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &StatusCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance().ErrorStreamOnly(),
	}
	c := &cobra.Command{
		Use:    "status",
		Short:  "Get the daemon status",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *StatusCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	socket := os.Getenv(platform.DaemonSocketEnv)
	if socket == "" {
		return fmt.Errorf("no socket for daemon found")
	}
	client := platformdaemon.NewClient(socket)
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}
	out, err := json.Marshal(status)
	if err != nil {
		return err
	}

	fmt.Print(string(out))

	return nil
}
