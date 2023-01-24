package tfexec

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

type taintConfig struct {
	state        string
	allowMissing bool
	lock         bool
	lockTimeout  string
}

var defaultTaintOptions = taintConfig{
	allowMissing: false,
	lock:         true,
}

// TaintOption represents options used in the Taint method.
type TaintOption interface {
	configureTaint(*taintConfig)
}

func (opt *StateOption) configureTaint(conf *taintConfig) {
	conf.state = opt.path
}

func (opt *AllowMissingOption) configureTaint(conf *taintConfig) {
	conf.allowMissing = opt.allowMissing
}

func (opt *LockOption) configureTaint(conf *taintConfig) {
	conf.lock = opt.lock
}

func (opt *LockTimeoutOption) configureTaint(conf *taintConfig) {
	conf.lockTimeout = opt.timeout
}

// Taint represents the terraform taint subcommand.
func (tf *Terraform) Taint(ctx context.Context, address string, opts ...TaintOption) error {
	err := tf.compatible(ctx, tf0_4_1, nil)
	if err != nil {
		return fmt.Errorf("taint was first introduced in Terraform 0.4.1: %w", err)
	}
	taintCmd := tf.taintCmd(ctx, address, opts...)
	return tf.runTerraformCmd(ctx, taintCmd)
}

func (tf *Terraform) taintCmd(ctx context.Context, address string, opts ...TaintOption) *exec.Cmd {
	c := defaultTaintOptions

	for _, o := range opts {
		o.configureTaint(&c)
	}

	args := []string{"taint", "-no-color"}

	if c.lockTimeout != "" {
		args = append(args, "-lock-timeout="+c.lockTimeout)
	}

	// string opts: only pass if set
	if c.state != "" {
		args = append(args, "-state="+c.state)
	}

	args = append(args, "-lock="+strconv.FormatBool(c.lock))
	if c.allowMissing {
		args = append(args, "-allow-missing")
	}
	args = append(args, address)

	return tf.buildTerraformCmd(ctx, nil, args...)
}
