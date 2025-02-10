package credentials

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"syscall"
	"time"

	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

var defaultBackoff = wait.Backoff{
	Steps:    4,
	Duration: 400 * time.Millisecond,
	Factor:   1,
	Jitter:   0.1,
}

func PostWithRetry(port int, endpoint string, body io.Reader, log log.Logger) ([]byte, error) {
	var out []byte
	err := retry.OnError(defaultBackoff, func(err error) bool {
		// connection refused is recoverable
		return errors.Is(err, syscall.ECONNREFUSED)
	}, func() error {
		url := fmt.Sprintf("http://localhost:%s/%s", strconv.Itoa(port), endpoint)
		response, err := devpodhttp.GetHTTPClient().Post(url, "application/json", body)
		if err != nil {
			log.Errorf("Error calling %s: %v", endpoint, err)
			return err
		}
		defer response.Body.Close()

		raw, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		// has the request succeeded?
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("call %s (%d): %s", endpoint, response.StatusCode, string(raw))
		}

		out = raw

		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}
