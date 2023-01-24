package tfexec

import (
	"context"
	"fmt"
	"os/exec"
)

type getCmdConfig struct {
	dir    string
	update bool
}

// GetCmdOption represents options used in the Get method.
type GetCmdOption interface {
	configureGet(*getCmdConfig)
}

func (opt *DirOption) configureGet(conf *getCmdConfig) {
	conf.dir = opt.path
}

func (opt *UpdateOption) configureGet(conf *getCmdConfig) {
	conf.update = opt.update
}

// Get represents the terraform get subcommand.
func (tf *Terraform) Get(ctx context.Context, opts ...GetCmdOption) error {
	cmd, err := tf.getCmd(ctx, opts...)
	if err != nil {
		return err
	}
	return tf.runTerraformCmd(ctx, cmd)
}

func (tf *Terraform) getCmd(ctx context.Context, opts ...GetCmdOption) (*exec.Cmd, error) {
	c := getCmdConfig{}

	for _, o := range opts {
		o.configureGet(&c)
	}

	args := []string{"get", "-no-color"}

	args = append(args, "-update="+fmt.Sprint(c.update))

	if c.dir != "" {
		args = append(args, c.dir)
	}

	return tf.buildTerraformCmd(ctx, nil, args...), nil
}
