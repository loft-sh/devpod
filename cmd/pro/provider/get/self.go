package get

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/platform/client"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// SelfCmd holds the cmd flags
type SelfCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewSelfCmd creates a new command
func NewSelfCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SelfCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "self",
		Short: "Get self",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *SelfCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	out, err := json.Marshal(baseClient.Self())
	if err != nil {
		return err
	}
	fmt.Println(string(out))

	return nil
}
