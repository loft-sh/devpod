package tfexec

import (
	"context"
	"fmt"
	"os/exec"
)

type providersLockConfig struct {
	fsMirror  string
	netMirror string
	platforms []string
	providers []string
}

var defaultProvidersLockOptions = providersLockConfig{}

type ProvidersLockOption interface {
	configureProvidersLock(*providersLockConfig)
}

func (opt *FSMirrorOption) configureProvidersLock(conf *providersLockConfig) {
	conf.fsMirror = opt.fsMirror
}

func (opt *NetMirrorOption) configureProvidersLock(conf *providersLockConfig) {
	conf.netMirror = opt.netMirror
}

func (opt *PlatformOption) configureProvidersLock(conf *providersLockConfig) {
	conf.platforms = append(conf.platforms, opt.platform)
}

func (opt *ProviderOption) configureProvidersLock(conf *providersLockConfig) {
	conf.providers = append(conf.providers, opt.provider)
}

// ProvidersLock represents the `terraform providers lock` command
func (tf *Terraform) ProvidersLock(ctx context.Context, opts ...ProvidersLockOption) error {
	err := tf.compatible(ctx, tf0_14_0, nil)
	if err != nil {
		return fmt.Errorf("terraform providers lock was added in 0.14.0: %w", err)
	}

	lockCmd := tf.providersLockCmd(ctx, opts...)

	err = tf.runTerraformCmd(ctx, lockCmd)
	if err != nil {
		return err
	}

	return err
}

func (tf *Terraform) providersLockCmd(ctx context.Context, opts ...ProvidersLockOption) *exec.Cmd {
	c := defaultProvidersLockOptions

	for _, o := range opts {
		o.configureProvidersLock(&c)
	}
	args := []string{"providers", "lock"}

	// string options, only pass if set
	if c.fsMirror != "" {
		args = append(args, "-fs-mirror="+c.fsMirror)
	}

	if c.netMirror != "" {
		args = append(args, "-net-mirror="+c.netMirror)
	}

	for _, p := range c.platforms {
		args = append(args, "-platform="+p)
	}

	// positional providers argument
	for _, p := range c.providers {
		args = append(args, p)
	}

	return tf.buildTerraformCmd(ctx, nil, args...)
}
