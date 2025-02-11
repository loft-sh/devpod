package ts

import (
	"context"
	"fmt"
	"os"

	"k8s.io/klog/v2"
	"tailscale.com/types/logger"
)

// TODO: adjust for DevPod
// TsnetLogger returns a logger that logs to klog if the LOFT_LOG_TSNET
// environment variable is set to true.
func TsnetLogger(ctx context.Context, serverName string) logger.Logf {
	logf := logger.Discard
	if os.Getenv("DEVPOD_LOG_TSNET") == "true" {
		logf = func(format string, args ...any) {
			klog.FromContext(ctx).V(1).Info("tsnet", "serverName", serverName, "msg", fmt.Sprintf(format, args...))
		}
	}
	return logf
}
