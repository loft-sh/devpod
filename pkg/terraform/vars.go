package terraform

import (
	"encoding/json"
	"os"
)

const VariablesFile = "terraform.tfvars.json"

func ReadVariables(file string, vars interface{}) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, vars)
	if err != nil {
		return err
	}

	return nil
}
