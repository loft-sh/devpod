package tfjson

import "encoding/json"

type unknownConstantValue struct{}

// UnknownConstantValue is a singleton type that denotes that a
// constant value is explicitly unknown. This is set during an
// unmarshal when references are found in an expression to help more
// explicitly differentiate between an explicit null and unknown
// value.
var UnknownConstantValue = &unknownConstantValue{}

// Expression describes the format for an individual key in a
// Terraform configuration.
//
// This struct wraps ExpressionData to support custom JSON parsing.
type Expression struct {
	*ExpressionData
}

// ExpressionData describes the format for an individual key in a
// Terraform configuration.
type ExpressionData struct {
	// If the *entire* expression is a constant-defined value, this
	// will contain the Go representation of the expression's data.
	//
	// Note that a nil here denotes and explicit null. When a value is
	// unknown on part of the value coming from an expression that
	// cannot be resolved at parse time, this field will contain
	// UnknownConstantValue.
	ConstantValue interface{} `json:"constant_value,omitempty"`

	// If any part of the expression contained values that were not
	// able to be resolved at parse-time, this will contain a list of
	// the referenced identifiers that caused the value to be unknown.
	References []string `json:"references,omitempty"`

	// A list of complex objects that were nested in this expression.
	// If this value is a nested block in configuration, sometimes
	// referred to as a "sub-resource", this field will contain those
	// values, and ConstantValue and References will be blank.
	NestedBlocks []map[string]*Expression `json:"-"`
}

// UnmarshalJSON implements json.Unmarshaler for Expression.
func (e *Expression) UnmarshalJSON(b []byte) error {
	result := new(ExpressionData)

	// Check to see if this is an array first. If it is, this is more
	// than likely a list of nested blocks.
	var rawNested []map[string]json.RawMessage
	if err := json.Unmarshal(b, &rawNested); err == nil {
		result.NestedBlocks, err = unmarshalExpressionBlocks(rawNested)
		if err != nil {
			return err
		}
	} else {
		// It's a non-nested expression block, parse normally
		if err := json.Unmarshal(b, &result); err != nil {
			return err
		}

		// If References is non-zero, then ConstantValue is unknown. Set
		// this explicitly.
		if len(result.References) > 0 {
			result.ConstantValue = UnknownConstantValue
		}
	}

	e.ExpressionData = result
	return nil
}

func unmarshalExpressionBlocks(raw []map[string]json.RawMessage) ([]map[string]*Expression, error) {
	var result []map[string]*Expression

	for _, rawBlock := range raw {
		block := make(map[string]*Expression)
		for k, rawExpr := range rawBlock {
			var expr *Expression
			if err := json.Unmarshal(rawExpr, &expr); err != nil {
				return nil, err
			}

			block[k] = expr
		}

		result = append(result, block)
	}

	return result, nil
}

// MarshalJSON implements json.Marshaler for Expression.
func (e *Expression) MarshalJSON() ([]byte, error) {
	switch {
	case len(e.ExpressionData.NestedBlocks) > 0:
		return marshalExpressionBlocks(e.ExpressionData.NestedBlocks)

	case e.ExpressionData.ConstantValue == UnknownConstantValue:
		return json.Marshal(&ExpressionData{
			References: e.ExpressionData.References,
		})
	}

	return json.Marshal(e.ExpressionData)
}

func marshalExpressionBlocks(nested []map[string]*Expression) ([]byte, error) {
	var rawNested []map[string]json.RawMessage
	for _, block := range nested {
		rawBlock := make(map[string]json.RawMessage)
		for k, expr := range block {
			raw, err := json.Marshal(expr)
			if err != nil {
				return nil, err
			}

			rawBlock[k] = raw
		}

		rawNested = append(rawNested, rawBlock)
	}

	return json.Marshal(rawNested)
}
