package platform

import (
	"os"
	"time"
)

const (
	defaultTimeout = 10 * time.Minute
)

func Timeout() time.Duration {
	if timeout := os.Getenv(TimeoutEnv); timeout != "" {
		if parsedTimeout, err := time.ParseDuration(timeout); err == nil {
			return parsedTimeout
		}
	}

	return defaultTimeout
}
