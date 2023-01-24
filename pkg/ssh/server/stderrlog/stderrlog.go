package stderrlog

import (
	"fmt"
	"io"
	"os"
)

var Writer io.Writer = os.Stderr

var debugModeEnabled = os.Getenv("DEVSPACE_HELPER_DEBUG") == "true"

func Errorf(message string, args ...interface{}) {
	_, _ = fmt.Fprintf(Writer, "error: "+message+"\n", args...)
}

func Infof(message string, args ...interface{}) {
	_, _ = fmt.Fprintf(Writer, message+"\n", args...)
}

func Debugf(message string, args ...interface{}) {
	if debugModeEnabled {
		_, _ = fmt.Fprintf(Writer, message+"\n", args...)
	}
}
