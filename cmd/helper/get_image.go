package helper

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/spf13/cobra"
)

type GetImageCommand struct {
	*flags.GlobalFlags
}

// NewGetImageCmd creates a new command
func NewGetImageCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetImageCommand{
		GlobalFlags: flags,
	}
	shellCmd := &cobra.Command{
		Use:   "get-image [image-name]",
		Short: "Retrieve details about an image",
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	return shellCmd
}

func (cmd *GetImageCommand) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("image name is missing")
	}

	img, err := image.GetImage(ctx, args[0])
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(img, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))

	return nil
}
