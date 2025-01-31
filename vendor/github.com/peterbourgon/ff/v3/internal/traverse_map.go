package internal

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// TraverseMap recursively walks the given map, calling set for each value. If the
// value is a slice, set is called for each element of the slice. The keys of
// nested maps are joined with the given delimiter.
func TraverseMap(m map[string]any, delimiter string, set func(name, value string) error) error {
	return traverseMap("", m, delimiter, set)
}

func traverseMap(key string, val any, delimiter string, set func(name, value string) error) error {
	switch v := val.(type) {
	case string:
		return set(key, v)
	case json.Number:
		return set(key, v.String())
	case uint64:
		return set(key, strconv.FormatUint(v, 10))
	case int:
		return set(key, strconv.Itoa(v))
	case int64:
		return set(key, strconv.FormatInt(v, 10))
	case float64:
		return set(key, strconv.FormatFloat(v, 'g', -1, 64))
	case bool:
		return set(key, strconv.FormatBool(v))
	case nil:
		return set(key, "")
	case []any:
		for _, v := range v {
			if err := traverseMap(key, v, delimiter, set); err != nil {
				return err
			}
		}
	case map[string]any:
		for k, v := range v {
			if key != "" {
				k = key + delimiter + k
			}
			if err := traverseMap(k, v, delimiter, set); err != nil {
				return err
			}
		}
	case map[any]any:
		for k, v := range v {
			ks := fmt.Sprint(k)
			if key != "" {
				ks = key + delimiter + ks
			}
			if err := traverseMap(ks, v, delimiter, set); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("couldn't convert %q (type %T) to string", val, val)
	}
	return nil
}
