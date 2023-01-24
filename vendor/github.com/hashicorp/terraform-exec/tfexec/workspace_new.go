package tfexec

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

type workspaceNewConfig struct {
	lock        bool
	lockTimeout string
	copyState   string
}

var defaultWorkspaceNewOptions = workspaceNewConfig{
	lock:        true,
	lockTimeout: "0s",
}

// WorkspaceNewCmdOption represents options that are applicable to the WorkspaceNew method.
type WorkspaceNewCmdOption interface {
	configureWorkspaceNew(*workspaceNewConfig)
}

func (opt *LockOption) configureWorkspaceNew(conf *workspaceNewConfig) {
	conf.lock = opt.lock
}

func (opt *LockTimeoutOption) configureWorkspaceNew(conf *workspaceNewConfig) {
	conf.lockTimeout = opt.timeout
}

func (opt *CopyStateOption) configureWorkspaceNew(conf *workspaceNewConfig) {
	conf.copyState = opt.path
}

// WorkspaceNew represents the workspace new subcommand to the Terraform CLI.
func (tf *Terraform) WorkspaceNew(ctx context.Context, workspace string, opts ...WorkspaceNewCmdOption) error {
	cmd, err := tf.workspaceNewCmd(ctx, workspace, opts...)
	if err != nil {
		return err
	}
	return tf.runTerraformCmd(ctx, cmd)
}

func (tf *Terraform) workspaceNewCmd(ctx context.Context, workspace string, opts ...WorkspaceNewCmdOption) (*exec.Cmd, error) {
	// TODO: [DIR] param option

	c := defaultWorkspaceNewOptions

	for _, o := range opts {
		switch o.(type) {
		case *LockOption, *LockTimeoutOption:
			err := tf.compatible(ctx, tf0_12_0, nil)
			if err != nil {
				return nil, fmt.Errorf("-lock and -lock-timeout were added to workspace new in Terraform 0.12: %w", err)
			}
		}

		o.configureWorkspaceNew(&c)
	}

	args := []string{"workspace", "new", "-no-color"}

	if c.lockTimeout != "" && c.lockTimeout != defaultWorkspaceNewOptions.lockTimeout {
		// only pass if not default, so we don't need to worry about the 0.11 version check
		args = append(args, "-lock-timeout="+c.lockTimeout)
	}
	if !c.lock {
		// only pass if false, so we don't need to worry about the 0.11 version check
		args = append(args, "-lock="+strconv.FormatBool(c.lock))
	}
	if c.copyState != "" {
		args = append(args, "-state="+c.copyState)
	}

	args = append(args, workspace)

	cmd := tf.buildTerraformCmd(ctx, nil, args...)

	return cmd, nil
}
