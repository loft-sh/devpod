package json

import (
	"encoding/json"
	"os"
)

func ReadFile(path string, out interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, out)
}
