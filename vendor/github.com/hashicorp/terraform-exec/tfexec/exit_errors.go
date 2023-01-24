package tfexec

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

// this file contains errors parsed from stderr

var (
	// The "Required variable not set:" case is for 0.11
	missingVarErrRegexp  = regexp.MustCompile(`Error: No value for required variable|Error: Required variable not set:`)
	missingVarNameRegexp = regexp.MustCompile(`The root module input variable\s"(.+)"\sis\snot\sset,\sand\shas\sno\sdefault|Error: Required variable not set: (.+)`)

	usageRegexp = regexp.MustCompile(`Too many command line arguments|^Usage: .*Options:.*|Error: Invalid -\d+ option`)

	noInitErrRegexp = regexp.MustCompile(
		// UNINITIALISED PROVIDERS/MODULES
		`Error: Could not satisfy plugin requirements|` +
			`Error: Could not load plugin|` + // v0.13
			`Please run \"terraform init\"|` + // v1.1.0 early alpha versions (ref 89b05050)
			`run:\s+terraform init|` + // v1.1.0 (ref df578afd)
			`Run\s+\"terraform init\"|` + // v1.2.0

			// UNINITIALISED BACKENDS
			`Error: Initialization required.|` + // v0.13
			`Error: Backend initialization required, please run \"terraform init\"`, // v0.15
	)

	noConfigErrRegexp = regexp.MustCompile(`Error: No configuration files`)

	workspaceDoesNotExistRegexp = regexp.MustCompile(`Workspace "(.+)" doesn't exist.`)

	workspaceAlreadyExistsRegexp = regexp.MustCompile(`Workspace "(.+)" already exists`)

	tfVersionMismatchErrRegexp        = regexp.MustCompile(`Error: The currently running version of Terraform doesn't meet the|Error: Unsupported Terraform Core version`)
	tfVersionMismatchConstraintRegexp = regexp.MustCompile(`required_version = "(.+)"|Required version: (.+)\b`)
	configInvalidErrRegexp            = regexp.MustCompile(`There are some problems with the configuration, described below.`)

	stateLockErrRegexp     = regexp.MustCompile(`Error acquiring the state lock`)
	stateLockInfoRegexp    = regexp.MustCompile(`Lock Info:\n\s*ID:\s*([^\n]+)\n\s*Path:\s*([^\n]+)\n\s*Operation:\s*([^\n]+)\n\s*Who:\s*([^\n]+)\n\s*Version:\s*([^\n]+)\n\s*Created:\s*([^\n]+)\n`)
	statePlanReadErrRegexp = regexp.MustCompile(
		`Terraform couldn't read the given file as a state or plan file.|` +
			`Error: Failed to read the given file as a state or plan file`)
	lockIdInvalidErrRegexp = regexp.MustCompile(`Failed to unlock state: `)
)

func (tf *Terraform) wrapExitError(ctx context.Context, err error, stderr string) error {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		// not an exit error, short circuit, nothing to wrap
		return err
	}

	ctxErr := ctx.Err()

	// nothing to parse, return early
	errString := strings.TrimSpace(stderr)
	if errString == "" {
		return &unwrapper{exitErr, ctxErr}
	}

	switch {
	case tfVersionMismatchErrRegexp.MatchString(stderr):
		constraint := ""
		constraints := tfVersionMismatchConstraintRegexp.FindStringSubmatch(stderr)
		for i := 1; i < len(constraints); i++ {
			constraint = strings.TrimSpace(constraints[i])
			if constraint != "" {
				break
			}
		}

		if constraint == "" {
			// hardcode a value here for weird cases (incl. 0.12)
			constraint = "unknown"
		}

		// only set this if it happened to be cached already
		ver := ""
		if tf != nil && tf.execVersion != nil {
			ver = tf.execVersion.String()
		}

		return &ErrTFVersionMismatch{
			unwrapper: unwrapper{exitErr, ctxErr},

			Constraint: constraint,
			TFVersion:  ver,
		}
	case missingVarErrRegexp.MatchString(stderr):
		name := ""
		names := missingVarNameRegexp.FindStringSubmatch(stderr)
		for i := 1; i < len(names); i++ {
			name = strings.TrimSpace(names[i])
			if name != "" {
				break
			}
		}

		return &ErrMissingVar{
			unwrapper: unwrapper{exitErr, ctxErr},

			VariableName: name,
		}
	case usageRegexp.MatchString(stderr):
		return &ErrCLIUsage{
			unwrapper: unwrapper{exitErr, ctxErr},

			stderr: stderr,
		}
	case noInitErrRegexp.MatchString(stderr):
		return &ErrNoInit{
			unwrapper: unwrapper{exitErr, ctxErr},

			stderr: stderr,
		}
	case noConfigErrRegexp.MatchString(stderr):
		return &ErrNoConfig{
			unwrapper: unwrapper{exitErr, ctxErr},

			stderr: stderr,
		}
	case workspaceDoesNotExistRegexp.MatchString(stderr):
		submatches := workspaceDoesNotExistRegexp.FindStringSubmatch(stderr)
		if len(submatches) == 2 {
			return &ErrNoWorkspace{
				unwrapper: unwrapper{exitErr, ctxErr},

				Name: submatches[1],
			}
		}
	case workspaceAlreadyExistsRegexp.MatchString(stderr):
		submatches := workspaceAlreadyExistsRegexp.FindStringSubmatch(stderr)
		if len(submatches) == 2 {
			return &ErrWorkspaceExists{
				unwrapper: unwrapper{exitErr, ctxErr},

				Name: submatches[1],
			}
		}
	case configInvalidErrRegexp.MatchString(stderr):
		return &ErrConfigInvalid{stderr: stderr}
	case stateLockErrRegexp.MatchString(stderr):
		submatches := stateLockInfoRegexp.FindStringSubmatch(stderr)
		if len(submatches) == 7 {
			return &ErrStateLocked{
				unwrapper: unwrapper{exitErr, ctxErr},

				ID:        submatches[1],
				Path:      submatches[2],
				Operation: submatches[3],
				Who:       submatches[4],
				Version:   submatches[5],
				Created:   submatches[6],
			}
		}
	case statePlanReadErrRegexp.MatchString(stderr):
		return &ErrStatePlanRead{stderr: stderr}
	case lockIdInvalidErrRegexp.MatchString(stderr):
		return &ErrLockIdInvalid{stderr: stderr}
	}

	return fmt.Errorf("%w\n%s", &unwrapper{exitErr, ctxErr}, stderr)
}

type unwrapper struct {
	err    error
	ctxErr error
}

func (u *unwrapper) Unwrap() error {
	return u.err
}

func (u *unwrapper) Is(target error) bool {
	switch target {
	case context.DeadlineExceeded, context.Canceled:
		return u.ctxErr == context.DeadlineExceeded ||
			u.ctxErr == context.Canceled
	}
	return false
}

func (u *unwrapper) Error() string {
	return u.err.Error()
}

type ErrConfigInvalid struct {
	stderr string
}

func (e *ErrConfigInvalid) Error() string {
	return "configuration is invalid"
}

type ErrMissingVar struct {
	unwrapper

	VariableName string
}

func (err *ErrMissingVar) Error() string {
	return fmt.Sprintf("variable %q was required but not supplied", err.VariableName)
}

type ErrNoWorkspace struct {
	unwrapper

	Name string
}

func (err *ErrNoWorkspace) Error() string {
	return fmt.Sprintf("workspace %q does not exist", err.Name)
}

// ErrWorkspaceExists is returned when creating a workspace that already exists
type ErrWorkspaceExists struct {
	unwrapper

	Name string
}

func (err *ErrWorkspaceExists) Error() string {
	return fmt.Sprintf("workspace %q already exists", err.Name)
}

type ErrNoInit struct {
	unwrapper

	stderr string
}

func (e *ErrNoInit) Error() string {
	return e.stderr
}

type ErrStatePlanRead struct {
	unwrapper

	stderr string
}

func (e *ErrStatePlanRead) Error() string {
	return e.stderr
}

type ErrNoConfig struct {
	unwrapper

	stderr string
}

func (e *ErrNoConfig) Error() string {
	return e.stderr
}

type ErrLockIdInvalid struct {
	unwrapper

	stderr string
}

func (e *ErrLockIdInvalid) Error() string {
	return e.stderr
}

// ErrCLIUsage is returned when the combination of flags or arguments is incorrect.
//
//  CLI indicates usage errors in three different ways: either
// 1. Exit 1, with a custom error message on stderr.
// 2. Exit 1, with command usage logged to stderr.
// 3. Exit 127, with command usage logged to stdout.
// Currently cases 1 and 2 are handled.
// TODO KEM: Handle exit 127 case. How does this work on non-Unix platforms?
type ErrCLIUsage struct {
	unwrapper

	stderr string
}

func (e *ErrCLIUsage) Error() string {
	return e.stderr
}

// ErrTFVersionMismatch is returned when the running Terraform version is not compatible with the
// value specified for required_version in the terraform block.
type ErrTFVersionMismatch struct {
	unwrapper

	TFVersion string

	// Constraint is not returned in the error messaging on 0.12
	Constraint string
}

func (e *ErrTFVersionMismatch) Error() string {
	version := "version"
	if e.TFVersion != "" {
		version = e.TFVersion
	}

	requirement := ""
	if e.Constraint != "" {
		requirement = fmt.Sprintf(" (%s required)", e.Constraint)
	}

	return fmt.Sprintf("terraform %s not supported by configuration%s",
		version, requirement)
}

// ErrStateLocked is returned when the state lock is already held by another process.
type ErrStateLocked struct {
	unwrapper

	ID        string
	Path      string
	Operation string
	Who       string
	Version   string
	Created   string
}

func (e *ErrStateLocked) Error() string {
	tmpl := `Lock Info:
  ID:        {{.ID}}
  Path:      {{.Path}}
  Operation: {{.Operation}}
  Who:       {{.Who}}
  Version:   {{.Version}}
  Created:   {{.Created}}
`

	t := template.Must(template.New("LockInfo").Parse(tmpl))
	var out strings.Builder
	if err := t.Execute(&out, e); err != nil {
		return "error acquiring the state lock"
	}
	return fmt.Sprintf("error acquiring the state lock: %v", out.String())
}
