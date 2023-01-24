package tfexec

import (
	"context"
	"fmt"
	"os/exec"
)

type forceUnlockConfig struct {
	dir string
}

var defaultForceUnlockOptions = forceUnlockConfig{}

type ForceUnlockOption interface {
	configureForceUnlock(*forceUnlockConfig)
}

func (opt *DirOption) configureForceUnlock(conf *forceUnlockConfig) {
	conf.dir = opt.path
}

// ForceUnlock represents the `terraform force-unlock` command
func (tf *Terraform) ForceUnlock(ctx context.Context, lockID string, opts ...ForceUnlockOption) error {
	unlockCmd, err := tf.forceUnlockCmd(ctx, lockID, opts...)
	if err != nil {
		return err
	}

	if err := tf.runTerraformCmd(ctx, unlockCmd); err != nil {
		return err
	}

	return nil
}

func (tf *Terraform) forceUnlockCmd(ctx context.Context, lockID string, opts ...ForceUnlockOption) (*exec.Cmd, error) {
	c := defaultForceUnlockOptions

	for _, o := range opts {
		o.configureForceUnlock(&c)
	}
	args := []string{"force-unlock", "-no-color", "-force"}

	// positional arguments
	args = append(args, lockID)

	// optional positional arguments
	if c.dir != "" {
		err := tf.compatible(ctx, nil, tf0_15_0)
		if err != nil {
			return nil, fmt.Errorf("[DIR] option was removed in Terraform v0.15.0")
		}
		args = append(args, c.dir)
	}

	return tf.buildTerraformCmd(ctx, nil, args...), nil
}
