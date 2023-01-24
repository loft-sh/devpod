package tfjson

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"
)

// ProviderSchemasFormatVersionConstraints defines the versions of the JSON
// provider schema format that are supported by this package.
var ProviderSchemasFormatVersionConstraints = ">= 0.1, < 2.0"

// ProviderSchemas represents the schemas of all providers and
// resources in use by the configuration.
type ProviderSchemas struct {
	// The version of the plan format. This should always match one of
	// ProviderSchemasFormatVersions in this package, or else
	// an unmarshal will be unstable.
	FormatVersion string `json:"format_version,omitempty"`

	// The schemas for the providers in this configuration, indexed by
	// provider type. Aliases are not included, and multiple instances
	// of a provider in configuration will be represented by a single
	// provider here.
	Schemas map[string]*ProviderSchema `json:"provider_schemas,omitempty"`
}

// Validate checks to ensure that ProviderSchemas is present, and the
// version matches the version supported by this library.
func (p *ProviderSchemas) Validate() error {
	if p == nil {
		return errors.New("provider schema data is nil")
	}

	if p.FormatVersion == "" {
		return errors.New("unexpected provider schema data, format version is missing")
	}

	constraint, err := version.NewConstraint(PlanFormatVersionConstraints)
	if err != nil {
		return fmt.Errorf("invalid version constraint: %w", err)
	}

	version, err := version.NewVersion(p.FormatVersion)
	if err != nil {
		return fmt.Errorf("invalid format version %q: %w", p.FormatVersion, err)
	}

	if !constraint.Check(version) {
		return fmt.Errorf("unsupported provider schema format version: %q does not satisfy %q",
			version, constraint)
	}

	return nil
}

func (p *ProviderSchemas) UnmarshalJSON(b []byte) error {
	type rawSchemas ProviderSchemas
	var schemas rawSchemas

	err := json.Unmarshal(b, &schemas)
	if err != nil {
		return err
	}

	*p = *(*ProviderSchemas)(&schemas)

	return p.Validate()
}

// ProviderSchema is the JSON representation of the schema of an
// entire provider, including the provider configuration and any
// resources and data sources included with the provider.
type ProviderSchema struct {
	// The schema for the provider's configuration.
	ConfigSchema *Schema `json:"provider,omitempty"`

	// The schemas for any resources in this provider.
	ResourceSchemas map[string]*Schema `json:"resource_schemas,omitempty"`

	// The schemas for any data sources in this provider.
	DataSourceSchemas map[string]*Schema `json:"data_source_schemas,omitempty"`
}

// Schema is the JSON representation of a particular schema
// (provider configuration, resources, data sources).
type Schema struct {
	// The version of the particular resource schema.
	Version uint64 `json:"version"`

	// The root-level block of configuration values.
	Block *SchemaBlock `json:"block,omitempty"`
}

// SchemaDescriptionKind describes the format type for a particular description's field.
type SchemaDescriptionKind string

const (
	// SchemaDescriptionKindPlain indicates a string in plain text format.
	SchemaDescriptionKindPlain SchemaDescriptionKind = "plain"

	// SchemaDescriptionKindMarkdown indicates a Markdown string and may need to be
	// processed prior to presentation.
	SchemaDescriptionKindMarkdown SchemaDescriptionKind = "markdown"
)

// SchemaBlock represents a nested block within a particular schema.
type SchemaBlock struct {
	// The attributes defined at the particular level of this block.
	Attributes map[string]*SchemaAttribute `json:"attributes,omitempty"`

	// Any nested blocks within this particular block.
	NestedBlocks map[string]*SchemaBlockType `json:"block_types,omitempty"`

	// The description for this block and format of the description. If
	// no kind is provided, it can be assumed to be plain text.
	Description     string                `json:"description,omitempty"`
	DescriptionKind SchemaDescriptionKind `json:"description_kind,omitempty"`

	// If true, this block is deprecated.
	Deprecated bool `json:"deprecated,omitempty"`
}

// SchemaNestingMode is the nesting mode for a particular nested
// schema block.
type SchemaNestingMode string

const (
	// SchemaNestingModeSingle denotes single block nesting mode, which
	// allows a single block of this specific type only in
	// configuration. This is generally the same as list or set types
	// with a single-element constraint.
	SchemaNestingModeSingle SchemaNestingMode = "single"

	// SchemaNestingModeGroup is similar to SchemaNestingModeSingle in that it
	// calls for only a single instance of a given block type with no labels,
	// but it additonally guarantees that its result will never be null,
	// even if the block is absent, and instead the nested attributes
	// and blocks will be treated as absent in that case.
	//
	// This is useful for the situation where a remote API has a feature that
	// is always enabled but has a group of settings related to that feature
	// that themselves have default values. By using SchemaNestingModeGroup
	// instead of SchemaNestingModeSingle in that case, generated plans will
	// show the block as present even when not present in configuration,
	// thus allowing any default values within to be displayed to the user.
	SchemaNestingModeGroup SchemaNestingMode = "group"

	// SchemaNestingModeList denotes list block nesting mode, which
	// allows an ordered list of blocks where duplicates are allowed.
	SchemaNestingModeList SchemaNestingMode = "list"

	// SchemaNestingModeSet denotes set block nesting mode, which
	// allows an unordered list of blocks where duplicates are
	// generally not allowed. What is considered a duplicate is up to
	// the rules of the set itself, which may or may not cover all
	// fields in the block.
	SchemaNestingModeSet SchemaNestingMode = "set"

	// SchemaNestingModeMap denotes map block nesting mode. This
	// creates a map of all declared blocks of the block type within
	// the parent, keying them on the label supplied in the block
	// declaration. This allows for blocks to be declared in the same
	// style as resources.
	SchemaNestingModeMap SchemaNestingMode = "map"
)

// SchemaBlockType describes a nested block within a schema.
type SchemaBlockType struct {
	// The nesting mode for this block.
	NestingMode SchemaNestingMode `json:"nesting_mode,omitempty"`

	// The block data for this block type, including attributes and
	// subsequent nested blocks.
	Block *SchemaBlock `json:"block,omitempty"`

	// The lower limit on items that can be declared of this block
	// type.
	MinItems uint64 `json:"min_items,omitempty"`

	// The upper limit on items that can be declared of this block
	// type.
	MaxItems uint64 `json:"max_items,omitempty"`
}

// SchemaAttribute describes an attribute within a schema block.
type SchemaAttribute struct {
	// The attribute type
	// Either AttributeType or AttributeNestedType is set, never both.
	AttributeType cty.Type `json:"type,omitempty"`

	// Details about a nested attribute type
	// Either AttributeType or AttributeNestedType is set, never both.
	AttributeNestedType *SchemaNestedAttributeType `json:"nested_type,omitempty"`

	// The description field for this attribute. If no kind is
	// provided, it can be assumed to be plain text.
	Description     string                `json:"description,omitempty"`
	DescriptionKind SchemaDescriptionKind `json:"description_kind,omitempty"`

	// If true, this attribute is deprecated.
	Deprecated bool `json:"deprecated,omitempty"`

	// If true, this attribute is required - it has to be entered in
	// configuration.
	Required bool `json:"required,omitempty"`

	// If true, this attribute is optional - it does not need to be
	// entered in configuration.
	Optional bool `json:"optional,omitempty"`

	// If true, this attribute is computed - it can be set by the
	// provider. It may also be set by configuration if Optional is
	// true.
	Computed bool `json:"computed,omitempty"`

	// If true, this attribute is sensitive and will not be displayed
	// in logs. Future versions of Terraform may encrypt or otherwise
	// treat these values with greater care than non-sensitive fields.
	Sensitive bool `json:"sensitive,omitempty"`
}

// jsonSchemaAttribute describes an attribute within a schema block
// in a middle-step internal representation before marshalled into
// a more useful SchemaAttribute with cty.Type.
//
// This avoid panic on marshalling cty.NilType (from cty upstream)
// which the default Go marshaller cannot ignore because it's a
// not nil-able struct.
type jsonSchemaAttribute struct {
	AttributeType       json.RawMessage            `json:"type,omitempty"`
	AttributeNestedType *SchemaNestedAttributeType `json:"nested_type,omitempty"`
	Description         string                     `json:"description,omitempty"`
	DescriptionKind     SchemaDescriptionKind      `json:"description_kind,omitempty"`
	Deprecated          bool                       `json:"deprecated,omitempty"`
	Required            bool                       `json:"required,omitempty"`
	Optional            bool                       `json:"optional,omitempty"`
	Computed            bool                       `json:"computed,omitempty"`
	Sensitive           bool                       `json:"sensitive,omitempty"`
}

func (as *SchemaAttribute) MarshalJSON() ([]byte, error) {
	jsonSa := &jsonSchemaAttribute{
		AttributeNestedType: as.AttributeNestedType,
		Description:         as.Description,
		DescriptionKind:     as.DescriptionKind,
		Deprecated:          as.Deprecated,
		Required:            as.Required,
		Optional:            as.Optional,
		Computed:            as.Computed,
		Sensitive:           as.Sensitive,
	}
	if as.AttributeType != cty.NilType {
		attrTy, _ := as.AttributeType.MarshalJSON()
		jsonSa.AttributeType = attrTy
	}
	return json.Marshal(jsonSa)
}

// SchemaNestedAttributeType describes a nested attribute
// which could also be just expressed simply as cty.Object(...),
// cty.List(cty.Object(...)) etc. but this allows tracking additional
// metadata which can help interpreting or validating the data.
type SchemaNestedAttributeType struct {
	// A map of nested attributes
	Attributes map[string]*SchemaAttribute `json:"attributes,omitempty"`

	// The nesting mode for this attribute.
	NestingMode SchemaNestingMode `json:"nesting_mode,omitempty"`

	// The lower limit on number of items that can be declared
	// of this attribute type (not applicable to single nesting mode).
	MinItems uint64 `json:"min_items,omitempty"`

	// The upper limit on number of items that can be declared
	// of this attribute type (not applicable to single nesting mode).
	MaxItems uint64 `json:"max_items,omitempty"`
}
