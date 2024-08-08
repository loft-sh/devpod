package loftconfig

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/log"
)

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

	response, err := devpodhttp.GetHTTPClient().Post(
		"http://localhost:"+strconv.Itoa(port)+"/loft-platform-credentials",
		"application/json",
		bytes.NewReader(rawJson),
	)
	if err != nil {
		logger.Errorf("Error retrieving credentials: %v", err)
		return nil, err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("Error reading loft config: %w", err)
		return nil, err
	}

	// has the request succeeded?
	if response.StatusCode != http.StatusOK {
		logger.Errorf("Error reading loft config (%d): %w", response.StatusCode, string(raw))
		return nil, err
	}

	configResponse := &LoftConfigResponse{}
	err = json.Unmarshal(raw, configResponse)
	if err != nil {
		logger.Errorf("Error decoding loft config: %s %w", string(raw), err)
		return nil, nil
	}

	return configResponse.LoftConfig, nil
}
