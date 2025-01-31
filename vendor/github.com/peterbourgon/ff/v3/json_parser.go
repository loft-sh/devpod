package ff

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/peterbourgon/ff/v3/internal"
)

// JSONParser is a helper function that uses a default JSONParseConfig.
func JSONParser(r io.Reader, set func(name, value string) error) error {
	return (&JSONParseConfig{}).Parse(r, set)
}

// JSONParseConfig collects parameters for the JSON config file parser.
type JSONParseConfig struct {
	// Delimiter is used when concatenating nested node keys into a flag name.
	// The default delimiter is ".".
	Delimiter string
}

// Parse a JSON document from the provided io.Reader, using the provided set
// function to set flag values. Flag names are derived from the node names and
// their key/value pairs.
func (pc *JSONParseConfig) Parse(r io.Reader, set func(name, value string) error) error {
	if pc.Delimiter == "" {
		pc.Delimiter = "."
	}

	d := json.NewDecoder(r)
	d.UseNumber() // required for stringifying values

	var m map[string]interface{}
	if err := d.Decode(&m); err != nil {
		return JSONParseError{Inner: err}
	}

	if err := internal.TraverseMap(m, pc.Delimiter, set); err != nil {
		return JSONParseError{Inner: err}
	}

	return nil
}

// JSONParseError wraps all errors originating from the JSONParser.
type JSONParseError struct {
	Inner error
}

// Error implenents the error interface.
func (e JSONParseError) Error() string {
	return fmt.Sprintf("error parsing JSON config: %v", e.Inner)
}

// Unwrap implements the errors.Wrapper interface, allowing errors.Is and
// errors.As to work with JSONParseErrors.
func (e JSONParseError) Unwrap() error {
	return e.Inner
}
