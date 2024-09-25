package loftconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"syscall"
	"time"

	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/log"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

var backoff = wait.Backoff{
	Steps:    4,
	Duration: 300 * time.Millisecond,
	Factor:   1,
	Jitter:   0.1,
}

func GetLoftConfig(context, provider string, port int, logger log.Logger) (*client.Config, error) {
	request := &LoftConfigRequest{
		Context:  context,
		Provider: provider,
	}

	rawJson, err := json.Marshal(request)
	if err != nil {
		logger.Errorf("Error parsing request: %w", err)
		return nil, err
	}

	configResponse := &LoftConfigResponse{}
	err = retry.OnError(backoff, func(err error) bool {
		// connection refused is recoverable
		return errors.Is(err, syscall.ECONNREFUSED)
	}, func() error {
		response, err := devpodhttp.GetHTTPClient().Post(
			"http://localhost:"+strconv.Itoa(port)+"/loft-platform-credentials",
			"application/json",
			bytes.NewReader(rawJson),
		)
		if err != nil {
			logger.Errorf("Error retrieving credentials: %v", err)
			return err
		}
		defer response.Body.Close()

		raw, err := io.ReadAll(response.Body)
		if err != nil {
			logger.Errorf("Error reading loft config: %w", err)
			return err
		}

		// has the request succeeded?
		if response.StatusCode != http.StatusOK {
			logger.Errorf("Error reading loft config (%d): %w", response.StatusCode, string(raw))
			return err
		}

		err = json.Unmarshal(raw, configResponse)
		if err != nil {
			logger.Errorf("Error decoding loft config: %s %w", string(raw), err)
			return nil
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return configResponse.LoftConfig, nil
}
