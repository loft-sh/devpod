package json

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/spf13/cobra"
)

type GetCmd struct {
	File string
	Fail bool
}

// NewGetCmd creates a new ssh command
func NewGetCmd() *cobra.Command {
	cmd := &GetCmd{}
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieves a JSON value by JSONPath",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	getCmd.Flags().StringVarP(&cmd.File, "file", "f", "", "Parse this json file instead of STDIN")
	getCmd.Flags().BoolVar(&cmd.Fail, "fail", false, "Fail if value is not found")
	return getCmd
}

func (cmd *GetCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("jsonpath expected")
	}

	if !strings.HasPrefix(args[0], "$") {
		if !strings.HasPrefix(args[0], "[") && !strings.HasPrefix(args[0], ".") {
			args[0] = "." + args[0]
		}

		args[0] = "$" + args[0]
	}

	var jsonBytes []byte
	if cmd.File != "" {
		var err error
		jsonBytes, err = os.ReadFile(cmd.File)
		if err != nil {
			return err
		}
	} else {
		var err error
		jsonBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	}

	v := interface{}(nil)
	err := json.Unmarshal(jsonBytes, &v)
	if err != nil {
		return fmt.Errorf("parse json")
	}

	val, err := jsonpath.Get(args[0], v)
	if err != nil {
		if cmd.Fail {
			return err
		}
		return nil
	}

	switch t := val.(type) {
	case string:
		fmt.Print(strings.TrimSpace(t))
		return nil
	case bool, int, int64, rune:
		fmt.Print(t)
		return nil
	}

	out, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}

	fmt.Print(string(out))
	return nil
}
