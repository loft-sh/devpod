package tfexec

import (
	"context"
	"os/exec"
	"strconv"
)

type importConfig struct {
	addr               string
	id                 string
	backup             string
	config             string
	allowMissingConfig bool
	lock               bool
	lockTimeout        string
	reattachInfo       ReattachInfo
	state              string
	stateOut           string
	vars               []string
	varFiles           []string
}

var defaultImportOptions = importConfig{
	allowMissingConfig: false,
	lock:               true,
	lockTimeout:        "0s",
}

// ImportOption represents options used in the Import method.
type ImportOption interface {
	configureImport(*importConfig)
}

func (opt *BackupOption) configureImport(conf *importConfig) {
	conf.backup = opt.path
}

func (opt *ConfigOption) configureImport(conf *importConfig) {
	conf.config = opt.path
}

func (opt *AllowMissingConfigOption) configureImport(conf *importConfig) {
	conf.allowMissingConfig = opt.allowMissingConfig
}

func (opt *LockOption) configureImport(conf *importConfig) {
	conf.lock = opt.lock
}

func (opt *LockTimeoutOption) configureImport(conf *importConfig) {
	conf.lockTimeout = opt.timeout
}

func (opt *ReattachOption) configureImport(conf *importConfig) {
	conf.reattachInfo = opt.info
}

func (opt *StateOption) configureImport(conf *importConfig) {
	conf.state = opt.path
}

func (opt *StateOutOption) configureImport(conf *importConfig) {
	conf.stateOut = opt.path
}

func (opt *VarOption) configureImport(conf *importConfig) {
	conf.vars = append(conf.vars, opt.assignment)
}

func (opt *VarFileOption) configureImport(conf *importConfig) {
	conf.varFiles = append(conf.varFiles, opt.path)
}

// Import represents the terraform import subcommand.
func (tf *Terraform) Import(ctx context.Context, address, id string, opts ...ImportOption) error {
	cmd, err := tf.importCmd(ctx, address, id, opts...)
	if err != nil {
		return err
	}
	return tf.runTerraformCmd(ctx, cmd)
}

func (tf *Terraform) importCmd(ctx context.Context, address, id string, opts ...ImportOption) (*exec.Cmd, error) {
	c := defaultImportOptions

	for _, o := range opts {
		o.configureImport(&c)
	}

	args := []string{"import", "-no-color", "-input=false"}

	// string opts: only pass if set
	if c.backup != "" {
		args = append(args, "-backup="+c.backup)
	}
	if c.config != "" {
		args = append(args, "-config="+c.config)
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
	for _, vf := range c.varFiles {
		args = append(args, "-var-file="+vf)
	}

	// boolean and numerical opts: always pass
	args = append(args, "-lock="+strconv.FormatBool(c.lock))

	// unary flags: pass if true
	if c.allowMissingConfig {
		args = append(args, "-allow-missing-config")
	}

	// string slice opts: split into separate args
	if c.vars != nil {
		for _, v := range c.vars {
			args = append(args, "-var", v)
		}
	}

	// required args, always pass
	args = append(args, address, id)

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
