package tfexec

import (
	"context"
	"fmt"
	"os/exec"
)

type initConfig struct {
	backend       bool
	backendConfig []string
	dir           string
	forceCopy     bool
	fromModule    string
	get           bool
	getPlugins    bool
	lock          bool
	lockTimeout   string
	pluginDir     []string
	reattachInfo  ReattachInfo
	reconfigure   bool
	upgrade       bool
	verifyPlugins bool
}

var defaultInitOptions = initConfig{
	backend:       true,
	forceCopy:     false,
	get:           true,
	getPlugins:    true,
	lock:          true,
	lockTimeout:   "0s",
	reconfigure:   false,
	upgrade:       false,
	verifyPlugins: true,
}

// InitOption represents options used in the Init method.
type InitOption interface {
	configureInit(*initConfig)
}

func (opt *BackendOption) configureInit(conf *initConfig) {
	conf.backend = opt.backend
}

func (opt *BackendConfigOption) configureInit(conf *initConfig) {
	conf.backendConfig = append(conf.backendConfig, opt.path)
}

func (opt *DirOption) configureInit(conf *initConfig) {
	conf.dir = opt.path
}

func (opt *ForceCopyOption) configureInit(conf *initConfig) {
	conf.forceCopy = opt.forceCopy
}

func (opt *FromModuleOption) configureInit(conf *initConfig) {
	conf.fromModule = opt.source
}

func (opt *GetOption) configureInit(conf *initConfig) {
	conf.get = opt.get
}

func (opt *GetPluginsOption) configureInit(conf *initConfig) {
	conf.getPlugins = opt.getPlugins
}

func (opt *LockOption) configureInit(conf *initConfig) {
	conf.lock = opt.lock
}

func (opt *LockTimeoutOption) configureInit(conf *initConfig) {
	conf.lockTimeout = opt.timeout
}

func (opt *PluginDirOption) configureInit(conf *initConfig) {
	conf.pluginDir = append(conf.pluginDir, opt.pluginDir)
}

func (opt *ReattachOption) configureInit(conf *initConfig) {
	conf.reattachInfo = opt.info
}

func (opt *ReconfigureOption) configureInit(conf *initConfig) {
	conf.reconfigure = opt.reconfigure
}

func (opt *UpgradeOption) configureInit(conf *initConfig) {
	conf.upgrade = opt.upgrade
}

func (opt *VerifyPluginsOption) configureInit(conf *initConfig) {
	conf.verifyPlugins = opt.verifyPlugins
}

// Init represents the terraform init subcommand.
func (tf *Terraform) Init(ctx context.Context, opts ...InitOption) error {
	cmd, err := tf.initCmd(ctx, opts...)
	if err != nil {
		return err
	}
	return tf.runTerraformCmd(ctx, cmd)
}

func (tf *Terraform) initCmd(ctx context.Context, opts ...InitOption) (*exec.Cmd, error) {
	c := defaultInitOptions

	for _, o := range opts {
		switch o.(type) {
		case *LockOption, *LockTimeoutOption, *VerifyPluginsOption, *GetPluginsOption:
			err := tf.compatible(ctx, nil, tf0_15_0)
			if err != nil {
				return nil, fmt.Errorf("-lock, -lock-timeout, -verify-plugins, and -get-plugins options are no longer available as of Terraform 0.15: %w", err)
			}
		}

		o.configureInit(&c)
	}

	args := []string{"init", "-no-color", "-input=false"}

	// string opts: only pass if set
	if c.fromModule != "" {
		args = append(args, "-from-module="+c.fromModule)
	}

	// string opts removed in 0.15: pass if set and <0.15
	err := tf.compatible(ctx, nil, tf0_15_0)
	if err == nil {
		if c.lockTimeout != "" {
			args = append(args, "-lock-timeout="+c.lockTimeout)
		}
	}

	// boolean opts: always pass
	args = append(args, "-backend="+fmt.Sprint(c.backend))
	args = append(args, "-get="+fmt.Sprint(c.get))
	args = append(args, "-upgrade="+fmt.Sprint(c.upgrade))

	// boolean opts removed in 0.15: pass if <0.15
	err = tf.compatible(ctx, nil, tf0_15_0)
	if err == nil {
		args = append(args, "-lock="+fmt.Sprint(c.lock))
		args = append(args, "-get-plugins="+fmt.Sprint(c.getPlugins))
		args = append(args, "-verify-plugins="+fmt.Sprint(c.verifyPlugins))
	}

	if c.forceCopy {
		args = append(args, "-force-copy")
	}

	// unary flags: pass if true
	if c.reconfigure {
		args = append(args, "-reconfigure")
	}

	// string slice opts: split into separate args
	if c.backendConfig != nil {
		for _, bc := range c.backendConfig {
			args = append(args, "-backend-config="+bc)
		}
	}
	if c.pluginDir != nil {
		for _, pd := range c.pluginDir {
			args = append(args, "-plugin-dir="+pd)
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
