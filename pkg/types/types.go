package types

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
)

var (
	// ErrUnsupportedType is returned if the type is not implemented
	ErrUnsupportedType = errors.New("unsupported type")
)

// StrIntArray string array to be used on JSON UnmarshalJSON
type StrIntArray []string

// UnmarshalJSON convert JSON object array of string or
// a string format strings to a golang string array
func (sa *StrIntArray) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return errors.Wrap(err, "unmarshal str int array")
	}
	switch obj := jsonObj.(type) {
	case string:
		*sa = StrIntArray([]string{obj})
		return nil
	case int:
		*sa = StrIntArray([]string{strconv.Itoa(obj)})
		return nil
	case []interface{}:
		s := make([]string, 0, len(obj))
		for _, v := range obj {
			switch value := v.(type) {
			case string:
				s = append(s, value)
			case int:
				s = append(s, strconv.Itoa(value))
			case float64:
				s = append(s, strconv.Itoa(int(value)))
			default:
				return ErrUnsupportedType
			}
		}
		*sa = StrIntArray(s)
		return nil
	}
	return ErrUnsupportedType
}

// StrArray string array to be used on JSON UnmarshalJSON
type StrArray []string

// UnmarshalJSON convert JSON object array of string or
// a string format strings to a golang string array
func (sa *StrArray) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	case string:
		*sa = StrArray([]string{obj})
		return nil
	case []interface{}:
		s := make([]string, 0, len(obj))
		for _, v := range obj {
			value, ok := v.(string)
			if !ok {
				return ErrUnsupportedType
			}
			s = append(s, value)
		}
		*sa = StrArray(s)
		return nil
	}
	return ErrUnsupportedType
}

type StrBool string

// UnmarshalJSON parses fields that may be numbers or booleans.
func (f *StrBool) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	case string:
		*f = StrBool(obj)
		return nil
	case bool:
		*f = StrBool(strconv.FormatBool(obj))
		return nil
	}
	return ErrUnsupportedType
}
