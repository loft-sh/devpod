package tfexec

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

type planConfig struct {
	destroy      bool
	dir          string
	lock         bool
	lockTimeout  string
	out          string
	parallelism  int
	reattachInfo ReattachInfo
	refresh      bool
	replaceAddrs []string
	state        string
	targets      []string
	vars         []string
	varFiles     []string
}

var defaultPlanOptions = planConfig{
	destroy:     false,
	lock:        true,
	lockTimeout: "0s",
	parallelism: 10,
	refresh:     true,
}

// PlanOption represents options used in the Plan method.
type PlanOption interface {
	configurePlan(*planConfig)
}

func (opt *DirOption) configurePlan(conf *planConfig) {
	conf.dir = opt.path
}

func (opt *VarFileOption) configurePlan(conf *planConfig) {
	conf.varFiles = append(conf.varFiles, opt.path)
}

func (opt *VarOption) configurePlan(conf *planConfig) {
	conf.vars = append(conf.vars, opt.assignment)
}

func (opt *TargetOption) configurePlan(conf *planConfig) {
	conf.targets = append(conf.targets, opt.target)
}

func (opt *StateOption) configurePlan(conf *planConfig) {
	conf.state = opt.path
}

func (opt *ReattachOption) configurePlan(conf *planConfig) {
	conf.reattachInfo = opt.info
}

func (opt *RefreshOption) configurePlan(conf *planConfig) {
	conf.refresh = opt.refresh
}

func (opt *ReplaceOption) configurePlan(conf *planConfig) {
	conf.replaceAddrs = append(conf.replaceAddrs, opt.address)
}

func (opt *ParallelismOption) configurePlan(conf *planConfig) {
	conf.parallelism = opt.parallelism
}

func (opt *OutOption) configurePlan(conf *planConfig) {
	conf.out = opt.path
}

func (opt *LockTimeoutOption) configurePlan(conf *planConfig) {
	conf.lockTimeout = opt.timeout
}

func (opt *LockOption) configurePlan(conf *planConfig) {
	conf.lock = opt.lock
}

func (opt *DestroyFlagOption) configurePlan(conf *planConfig) {
	conf.destroy = opt.destroy
}

// Plan executes `terraform plan` with the specified options and waits for it
// to complete.
//
// The returned boolean is false when the plan diff is empty (no changes) and
// true when the plan diff is non-empty (changes present).
//
// The returned error is nil if `terraform plan` has been executed and exits
// with either 0 or 2.
func (tf *Terraform) Plan(ctx context.Context, opts ...PlanOption) (bool, error) {
	cmd, err := tf.planCmd(ctx, opts...)
	if err != nil {
		return false, err
	}
	err = tf.runTerraformCmd(ctx, cmd)
	if err != nil && cmd.ProcessState.ExitCode() == 2 {
		return true, nil
	}
	return false, err
}

func (tf *Terraform) planCmd(ctx context.Context, opts ...PlanOption) (*exec.Cmd, error) {
	c := defaultPlanOptions

	for _, o := range opts {
		o.configurePlan(&c)
	}

	args := []string{"plan", "-no-color", "-input=false", "-detailed-exitcode"}

	// string opts: only pass if set
	if c.lockTimeout != "" {
		args = append(args, "-lock-timeout="+c.lockTimeout)
	}
	if c.out != "" {
		args = append(args, "-out="+c.out)
	}
	if c.state != "" {
		args = append(args, "-state="+c.state)
	}
	for _, vf := range c.varFiles {
		args = append(args, "-var-file="+vf)
	}

	// boolean and numerical opts: always pass
	args = append(args, "-lock="+strconv.FormatBool(c.lock))
	args = append(args, "-parallelism="+fmt.Sprint(c.parallelism))
	args = append(args, "-refresh="+strconv.FormatBool(c.refresh))

	// unary flags: pass if true
	if c.replaceAddrs != nil {
		err := tf.compatible(ctx, tf0_15_2, nil)
		if err != nil {
			return nil, fmt.Errorf("replace option was introduced in Terraform 0.15.2: %w", err)
		}
		for _, addr := range c.replaceAddrs {
			args = append(args, "-replace="+addr)
		}
	}
	if c.destroy {
		args = append(args, "-destroy")
	}

	// string slice opts: split into separate args
	if c.targets != nil {
		for _, ta := range c.targets {
			args = append(args, "-target="+ta)
		}
	}
	if c.vars != nil {
		for _, v := range c.vars {
			args = append(args, "-var", v)
		}
	}

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
