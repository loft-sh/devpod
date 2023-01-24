package tfexec

import (
	"context"
	"os/exec"
	"strconv"
)

type stateMvConfig struct {
	backup      string
	backupOut   string
	dryRun      bool
	lock        bool
	lockTimeout string
	state       string
	stateOut    string
}

var defaultStateMvOptions = stateMvConfig{
	lock:        true,
	lockTimeout: "0s",
}

// StateMvCmdOption represents options used in the Refresh method.
type StateMvCmdOption interface {
	configureStateMv(*stateMvConfig)
}

func (opt *BackupOption) configureStateMv(conf *stateMvConfig) {
	conf.backup = opt.path
}

func (opt *BackupOutOption) configureStateMv(conf *stateMvConfig) {
	conf.backupOut = opt.path
}

func (opt *DryRunOption) configureStateMv(conf *stateMvConfig) {
	conf.dryRun = opt.dryRun
}

func (opt *LockOption) configureStateMv(conf *stateMvConfig) {
	conf.lock = opt.lock
}

func (opt *LockTimeoutOption) configureStateMv(conf *stateMvConfig) {
	conf.lockTimeout = opt.timeout
}

func (opt *StateOption) configureStateMv(conf *stateMvConfig) {
	conf.state = opt.path
}

func (opt *StateOutOption) configureStateMv(conf *stateMvConfig) {
	conf.stateOut = opt.path
}

// StateMv represents the terraform state mv subcommand.
func (tf *Terraform) StateMv(ctx context.Context, source string, destination string, opts ...StateMvCmdOption) error {
	cmd, err := tf.stateMvCmd(ctx, source, destination, opts...)
	if err != nil {
		return err
	}
	return tf.runTerraformCmd(ctx, cmd)
}

func (tf *Terraform) stateMvCmd(ctx context.Context, source string, destination string, opts ...StateMvCmdOption) (*exec.Cmd, error) {
	c := defaultStateMvOptions

	for _, o := range opts {
		o.configureStateMv(&c)
	}

	args := []string{"state", "mv", "-no-color"}

	// string opts: only pass if set
	if c.backup != "" {
		args = append(args, "-backup="+c.backup)
	}
	if c.backupOut != "" {
		args = append(args, "-backup-out="+c.backupOut)
	}
	if c.lockTimeout != "" {
		args = append(args, "-lock-timeout="+c.lockTimeout)
	}
	if c.state != "" {
		args = append(args, "-state="+c.state)
	}
	if c.stateOut != "" {
		args = append(args, "-state-out="+c.stateOut)
	}

	// boolean and numerical opts: always pass
	args = append(args, "-lock="+strconv.FormatBool(c.lock))

	// unary flags: pass if true
	if c.dryRun {
		args = append(args, "-dry-run")
	}

	// positional arguments
	args = append(args, source)
	args = append(args, destination)

	return tf.buildTerraformCmd(ctx, nil, args...), nil
}
