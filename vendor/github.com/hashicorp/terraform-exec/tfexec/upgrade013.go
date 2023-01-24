package tfexec

import (
	"context"
	"fmt"
	"os/exec"
)

type upgrade013Config struct {
	dir string

	reattachInfo ReattachInfo
}

var defaultUpgrade013Options = upgrade013Config{}

// Upgrade013Option represents options used in the Destroy method.
type Upgrade013Option interface {
	configureUpgrade013(*upgrade013Config)
}

func (opt *DirOption) configureUpgrade013(conf *upgrade013Config) {
	conf.dir = opt.path
}

func (opt *ReattachOption) configureUpgrade013(conf *upgrade013Config) {
	conf.reattachInfo = opt.info
}

// Upgrade013 represents the terraform 0.13upgrade subcommand.
func (tf *Terraform) Upgrade013(ctx context.Context, opts ...Upgrade013Option) error {
	cmd, err := tf.upgrade013Cmd(ctx, opts...)
	if err != nil {
		return err
	}
	return tf.runTerraformCmd(ctx, cmd)
}

func (tf *Terraform) upgrade013Cmd(ctx context.Context, opts ...Upgrade013Option) (*exec.Cmd, error) {
	err := tf.compatible(ctx, tf0_13_0, tf0_14_0)
	if err != nil {
		return nil, fmt.Errorf("terraform 0.13upgrade is only supported in 0.13 releases: %w", err)
	}

	c := defaultUpgrade013Options

	for _, o := range opts {
		o.configureUpgrade013(&c)
	}

	args := []string{"0.13upgrade", "-no-color", "-yes"}

	// optional positional argument
	if c.dir != "" {
		args = append(args, c.dir)
	}

	mergeEnv := map[string]string{}
	if c.reattachInfo != nil {
		reattachStr, err := c.reattachInfo.marshalString()
		if err != nil {
			return nil, err
		}
		mergeEnv[reattachEnvVar] = reattachStr
	}

	return tf.buildTerraformCmd(ctx, mergeEnv, args...), nil
}
