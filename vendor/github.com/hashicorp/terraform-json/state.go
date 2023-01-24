package tfjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"
)

// StateFormatVersionConstraints defines the versions of the JSON state format
// that are supported by this package.
var StateFormatVersionConstraints = ">= 0.1, < 2.0"

// State is the top-level representation of a Terraform state.
type State struct {
	// useJSONNumber opts into the behavior of calling
	// json.Decoder.UseNumber prior to decoding the state, which turns
	// numbers into json.Numbers instead of float64s. Set it using
	// State.UseJSONNumber.
	useJSONNumber bool

	// The version of the state format. This should always match the
	// StateFormatVersion constant in this package, or else am
	// unmarshal will be unstable.
	FormatVersion string `json:"format_version,omitempty"`

	// The Terraform version used to make the state.
	TerraformVersion string `json:"terraform_version,omitempty"`

	// The values that make up the state.
	Values *StateValues `json:"values,omitempty"`
}

// UseJSONNumber controls whether the State will be decoded using the
// json.Number behavior or the float64 behavior. When b is true, the State will
// represent numbers in StateOutputs as json.Numbers. When b is false, the
// State will represent numbers in StateOutputs as float64s.
func (s *State) UseJSONNumber(b bool) {
	s.useJSONNumber = b
}

// Validate checks to ensure that the state is present, and the
// version matches the version supported by this library.
func (s *State) Validate() error {
	if s == nil {
		return errors.New("state is nil")
	}

	if s.FormatVersion == "" {
		return errors.New("unexpected state input, format version is missing")
	}

	constraint, err := version.NewConstraint(StateFormatVersionConstraints)
	if err != nil {
		return fmt.Errorf("invalid version constraint: %w", err)
	}

	version, err := version.NewVersion(s.FormatVersion)
	if err != nil {
		return fmt.Errorf("invalid format version %q: %w", s.FormatVersion, err)
	}

	if !constraint.Check(version) {
		return fmt.Errorf("unsupported state format version: %q does not satisfy %q",
			version, constraint)
	}

	return nil
}

func (s *State) UnmarshalJSON(b []byte) error {
	type rawState State
	var state rawState

	dec := json.NewDecoder(bytes.NewReader(b))
	if s.useJSONNumber {
		dec.UseNumber()
	}
	err := dec.Decode(&state)
	if err != nil {
		return err
	}

	*s = *(*State)(&state)

	return s.Validate()
}

// StateValues is the common representation of resolved values for both the
// prior state (which is always complete) and the planned new state.
type StateValues struct {
	// The Outputs for this common state representation.
	Outputs map[string]*StateOutput `json:"outputs,omitempty"`

	// The root module in this state representation.
	RootModule *StateModule `json:"root_module,omitempty"`
}

// StateModule is the representation of a module in the common state
// representation. This can be the root module or a child module.
type StateModule struct {
	// All resources or data sources within this module.
	Resources []*StateResource `json:"resources,omitempty"`

	// The absolute module address, omitted for the root module.
	Address string `json:"address,omitempty"`

	// Any child modules within this module.
	ChildModules []*StateModule `json:"child_modules,omitempty"`
}

// StateResource is the representation of a resource in the common
// state representation.
type StateResource struct {
	// The absolute resource address.
	Address string `json:"address,omitempty"`

	// The resource mode.
	Mode ResourceMode `json:"mode,omitempty"`

	// The resource type, example: "aws_instance" for aws_instance.foo.
	Type string `json:"type,omitempty"`

	// The resource name, example: "foo" for aws_instance.foo.
	Name string `json:"name,omitempty"`

	// The instance key for any resources that have been created using
	// "count" or "for_each". If neither of these apply the key will be
	// empty.
	//
	// This value can be either an integer (int) or a string.
	Index interface{} `json:"index,omitempty"`

	// The name of the provider this resource belongs to. This allows
	// the provider to be interpreted unambiguously in the unusual
	// situation where a provider offers a resource type whose name
	// does not start with its own name, such as the "googlebeta"
	// provider offering "google_compute_instance".
	ProviderName string `json:"provider_name,omitempty"`

	// The version of the resource type schema the "values" property
	// conforms to.
	SchemaVersion uint64 `json:"schema_version,"`

	// The JSON representation of the attribute values of the resource,
	// whose structure depends on the resource type schema. Any unknown
	// values are omitted or set to null, making them indistinguishable
	// from absent values.
	AttributeValues map[string]interface{} `json:"values,omitempty"`

	// The JSON representation of the sensitivity of the resource's
	// attribute values. Only attributes which are sensitive
	// are included in this structure.
	SensitiveValues json.RawMessage `json:"sensitive_values,omitempty"`

	// The addresses of the resources that this resource depends on.
	DependsOn []string `json:"depends_on,omitempty"`

	// If true, the resource has been marked as tainted and will be
	// re-created on the next update.
	Tainted bool `json:"tainted,omitempty"`

	// DeposedKey is set if the resource instance has been marked Deposed and
	// will be destroyed on the next apply.
	DeposedKey string `json:"deposed_key,omitempty"`
}

// StateOutput represents an output value in a common state
// representation.
type StateOutput struct {
	// Whether or not the output was marked as sensitive.
	Sensitive bool `json:"sensitive"`

	// The value of the output.
	Value interface{} `json:"value,omitempty"`

	// The type of the output.
	Type cty.Type `json:"type,omitempty"`
}

// jsonStateOutput describes an output value in a middle-step internal
// representation before marshalled into a more useful StateOutput with cty.Type.
//
// This avoid panic on marshalling cty.NilType (from cty upstream)
// which the default Go marshaller cannot ignore because it's a
// not nil-able struct.
type jsonStateOutput struct {
	Sensitive bool            `json:"sensitive"`
	Value     interface{}     `json:"value,omitempty"`
	Type      json.RawMessage `json:"type,omitempty"`
}

func (so *StateOutput) MarshalJSON() ([]byte, error) {
	jsonSa := &jsonStateOutput{
		Sensitive: so.Sensitive,
		Value:     so.Value,
	}
	if so.Type != cty.NilType {
		outputType, _ := so.Type.MarshalJSON()
		jsonSa.Type = outputType
	}
	return json.Marshal(jsonSa)
}
