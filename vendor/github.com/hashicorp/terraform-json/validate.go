package tfjson

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/go-version"
)

// ValidateFormatVersionConstraints defines the versions of the JSON
// validate format that are supported by this package.
var ValidateFormatVersionConstraints = ">= 0.1, < 2.0"

// Pos represents a position in a config file
type Pos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Byte   int `json:"byte"`
}

// Range represents a range of bytes between two positions
type Range struct {
	Filename string `json:"filename"`
	Start    Pos    `json:"start"`
	End      Pos    `json:"end"`
}

type DiagnosticSeverity string

// These severities map to the tfdiags.Severity values, plus an explicit
// unknown in case that enum grows without us noticing here.
const (
	DiagnosticSeverityUnknown DiagnosticSeverity = "unknown"
	DiagnosticSeverityError   DiagnosticSeverity = "error"
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
)

// Diagnostic represents information to be presented to a user about an
// error or anomaly in parsing or evaluating configuration
type Diagnostic struct {
	Severity DiagnosticSeverity `json:"severity,omitempty"`

	Summary string `json:"summary,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Range   *Range `json:"range,omitempty"`

	Snippet *DiagnosticSnippet `json:"snippet,omitempty"`
}

// DiagnosticSnippet represents source code information about the diagnostic.
// It is possible for a diagnostic to have a source (and therefore a range) but
// no source code can be found. In this case, the range field will be present and
// the snippet field will not.
type DiagnosticSnippet struct {
	// Context is derived from HCL's hcled.ContextString output. This gives a
	// high-level summary of the root context of the diagnostic: for example,
	// the resource block in which an expression causes an error.
	Context *string `json:"context"`

	// Code is a possibly-multi-line string of Terraform configuration, which
	// includes both the diagnostic source and any relevant context as defined
	// by the diagnostic.
	Code string `json:"code"`

	// StartLine is the line number in the source file for the first line of
	// the snippet code block. This is not necessarily the same as the value of
	// Range.Start.Line, as it is possible to have zero or more lines of
	// context source code before the diagnostic range starts.
	StartLine int `json:"start_line"`

	// HighlightStartOffset is the character offset into Code at which the
	// diagnostic source range starts, which ought to be highlighted as such by
	// the consumer of this data.
	HighlightStartOffset int `json:"highlight_start_offset"`

	// HighlightEndOffset is the character offset into Code at which the
	// diagnostic source range ends.
	HighlightEndOffset int `json:"highlight_end_offset"`

	// Values is a sorted slice of expression values which may be useful in
	// understanding the source of an error in a complex expression.
	Values []DiagnosticExpressionValue `json:"values"`
}

// DiagnosticExpressionValue represents an HCL traversal string (e.g.
// "var.foo") and a statement about its value while the expression was
// evaluated (e.g. "is a string", "will be known only after apply"). These are
// intended to help the consumer diagnose why an expression caused a diagnostic
// to be emitted.
type DiagnosticExpressionValue struct {
	Traversal string `json:"traversal"`
	Statement string `json:"statement"`
}

// ValidateOutput represents JSON output from terraform validate
// (available from 0.12 onwards)
type ValidateOutput struct {
	FormatVersion string `json:"format_version"`

	Valid        bool         `json:"valid"`
	ErrorCount   int          `json:"error_count"`
	WarningCount int          `json:"warning_count"`
	Diagnostics  []Diagnostic `json:"diagnostics"`
}

// Validate checks to ensure that data is present, and the
// version matches the version supported by this library.
func (vo *ValidateOutput) Validate() error {
	if vo == nil {
		return errors.New("validation output is nil")
	}

	if vo.FormatVersion == "" {
		// The format was not versioned in the past
		return nil
	}

	constraint, err := version.NewConstraint(ValidateFormatVersionConstraints)
	if err != nil {
		return fmt.Errorf("invalid version constraint: %w", err)
	}

	version, err := version.NewVersion(vo.FormatVersion)
	if err != nil {
		return fmt.Errorf("invalid format version %q: %w", vo.FormatVersion, err)
	}

	if !constraint.Check(version) {
		return fmt.Errorf("unsupported validation output format version: %q does not satisfy %q",
			version, constraint)
	}

	return nil
}

func (vo *ValidateOutput) UnmarshalJSON(b []byte) error {
	type rawOutput ValidateOutput
	var schemas rawOutput

	err := json.Unmarshal(b, &schemas)
	if err != nil {
		return err
	}

	*vo = *(*ValidateOutput)(&schemas)

	return vo.Validate()
}
