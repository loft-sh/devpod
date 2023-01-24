package tfexec

import (
	"context"
	"os/exec"

	tfjson "github.com/hashicorp/terraform-json"
)

// ProvidersSchema represents the terraform providers schema -json subcommand.
func (tf *Terraform) ProvidersSchema(ctx context.Context) (*tfjson.ProviderSchemas, error) {
	schemaCmd := tf.providersSchemaCmd(ctx)

	var ret tfjson.ProviderSchemas
	err := tf.runTerraformCmdJSON(ctx, schemaCmd, &ret)
	if err != nil {
		return nil, err
	}

	err = ret.Validate()
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (tf *Terraform) providersSchemaCmd(ctx context.Context, args ...string) *exec.Cmd {
	allArgs := []string{"providers", "schema", "-json", "-no-color"}
	allArgs = append(allArgs, args...)

	return tf.buildTerraformCmd(ctx, nil, allArgs...)
}
