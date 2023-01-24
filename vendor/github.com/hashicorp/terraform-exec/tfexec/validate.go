package tfexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
)

// Validate represents the validate subcommand to the Terraform CLI. The -json
// flag support was added in 0.12.0, so this will not work on earlier versions.
func (tf *Terraform) Validate(ctx context.Context) (*tfjson.ValidateOutput, error) {
	err := tf.compatible(ctx, tf0_12_0, nil)
	if err != nil {
		return nil, fmt.Errorf("terraform validate -json was added in 0.12.0: %w", err)
	}

	cmd := tf.buildTerraformCmd(ctx, nil, "validate", "-no-color", "-json")

	var outBuf = bytes.Buffer{}
	cmd.Stdout = &outBuf

	err = tf.runTerraformCmd(ctx, cmd)
	// TODO: this command should not exit 1 if you pass -json as its hard to differentiate other errors
	if err != nil && cmd.ProcessState.ExitCode() != 1 {
		return nil, err
	}

	var ret tfjson.ValidateOutput
	// TODO: ret.UseJSONNumber(true) validate output should support JSON numbers
	jsonErr := json.Unmarshal(outBuf.Bytes(), &ret)
	if jsonErr != nil {
		// the original call was possibly bad, if it has an error, actually just return that
		if err != nil {
			return nil, err
		}

		return nil, jsonErr
	}

	return &ret, nil
}
