package tfexec

import "fmt"

// this file contains non-parsed exported errors

type ErrNoSuitableBinary struct {
	err error
}

func (e *ErrNoSuitableBinary) Error() string {
	return fmt.Sprintf("no suitable terraform binary could be found: %s", e.err.Error())
}

func (e *ErrNoSuitableBinary) Unwrap() error {
	return e.err
}

// ErrVersionMismatch is returned when the detected Terraform version is not compatible with the
// command or flags being used in this invocation.
type ErrVersionMismatch struct {
	MinInclusive string
	MaxExclusive string
	Actual       string
}

func (e *ErrVersionMismatch) Error() string {
	return fmt.Sprintf("unexpected version %s (min: %s, max: %s)", e.Actual, e.MinInclusive, e.MaxExclusive)
}

// ErrManualEnvVar is returned when an env var that should be set programatically via an option or method
// is set via the manual environment passing functions.
type ErrManualEnvVar struct {
	Name string
}

func (err *ErrManualEnvVar) Error() string {
	return fmt.Sprintf("manual setting of env var %q detected", err.Name)
}
