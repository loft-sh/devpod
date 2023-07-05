package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RequestCmd struct {
	Method  string
	Data    string
	Headers []string

	FailOnErrorCode bool
}

// NewRequestCmd creates a new ssh command
func NewRequestCmd() *cobra.Command {
	cmd := &RequestCmd{}
	requestCmd := &cobra.Command{
		Use:   "request",
		Short: "Executes a http(s) request",
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run(context.Background(), args)
		},
	}

	requestCmd.Flags().StringVarP(&cmd.Data, "data", "d", "", "Request Data")
	requestCmd.Flags().StringVarP(&cmd.Method, "request", "X", "GET", "Request Type")
	requestCmd.Flags().StringSliceVarP(&cmd.Headers, "header", "H", []string{}, "Extra Headers")
	requestCmd.Flags().BoolVar(&cmd.FailOnErrorCode, "fail-on-error-code", true, "Let this command fail if the remote is returning an error code")
	return requestCmd
}

func (cmd *RequestCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected request url as argument")
	}

	cmd.Method = strings.ToUpper(cmd.Method)
	httpHeader := http.Header{}
	for _, header := range cmd.Headers {
		splitted := strings.Split(header, ":")
		if len(splitted) == 1 {
			return fmt.Errorf("unexpected header '%s', expected form 'HEADER: VALUE'", header)
		}

		httpHeader.Add(strings.TrimSpace(splitted[0]), strings.TrimSpace(strings.Join(splitted[1:], ":")))
	}

	request, err := http.NewRequest(cmd.Method, args[0], strings.NewReader(cmd.Data))
	if err != nil {
		return err
	}
	request.Header = httpHeader

	resp, err := devpodhttp.GetHTTPClient().Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		out, _ := io.ReadAll(resp.Body)
		_, _ = fmt.Fprint(os.Stderr, string(out))
		return fmt.Errorf("unexpected response code %d", resp.StatusCode)
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return errors.Wrap(err, "read response")
	}

	return nil
}
