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
	case float64:
		*sa = StrIntArray([]string{strconv.Itoa(int(obj))})
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

type LifecycleHook map[string][]string

func (l *LifecycleHook) UnmarshalJSON(data []byte) error {
	*l = make(map[string][]string)

	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	case string:
		// Anonymous string command
		(*l)[""] = []string{obj}
		return nil
	case []interface{}:
		// Anonymous array of strings command
		cmd := make([]string, 0)
		for _, v := range obj {
			value, ok := v.(string)
			if !ok {
				return ErrUnsupportedType
			}
			cmd = append(cmd, value)
		}
		(*l)[""] = cmd
		return nil
	case map[string]interface{}:
		for k, v := range obj {
			value, ok := v.(string)
			if ok {
				// Named string command
				(*l)[k] = []string{value}
			} else {
				// Named array of strings command
				stringArrayValue, ok := v.([]interface{})
				if !ok {
					return ErrUnsupportedType
				}

				cmd := make([]string, 0)
				for _, v := range stringArrayValue {
					cmd = append(cmd, v.(string))
				}
				(*l)[k] = cmd
			}
		}

		return nil
	}

	return ErrUnsupportedType
}

type StrBool string

// UnmarshalJSON parses fields that may be numbers or booleans.
func (s *StrBool) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	case string:
		*s = StrBool(obj)
		return nil
	case bool:
		*s = StrBool(strconv.FormatBool(obj))
		return nil
	}
	return ErrUnsupportedType
}

func (s *StrBool) Bool() (bool, error) {
	if s == nil {
		return false, nil
	}

	return strconv.ParseBool(string(*s))
}

type OptionEnum struct {
	Value       string `json:"value,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type OptionEnumArray []OptionEnum

func (e *OptionEnumArray) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	case []interface{}:
		if len(obj) == 0 {
			*e = OptionEnumArray{}
			return nil
		}
		ret := make([]OptionEnum, 0, len(obj))
		switch obj[0].(type) {
		case string:
			for _, v := range obj {
				if s, ok := v.(string); ok {
					ret = append(ret, OptionEnum{Value: s})
				}
			}
		case map[string]interface{}:
			for _, v := range obj {
				m, ok := v.(map[string]interface{})
				if !ok {
					return ErrUnsupportedType
				}
				value := ""
				if s, ok := m["value"].(string); ok {
					value = s
				}
				displayName := ""
				if s, ok := m["displayName"].(string); ok {
					displayName = s
				}
				ret = append(ret, OptionEnum{
					Value:       value,
					DisplayName: displayName,
				})
			}
		default:
			return ErrUnsupportedType
		}

		*e = OptionEnumArray(ret)
		return nil
	}

	return ErrUnsupportedType
}
