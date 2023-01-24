package tfexec

import (
	"bytes"
	"context"
	"os/exec"
)

type statePullConfig struct {
	reattachInfo ReattachInfo
}

var defaultStatePullConfig = statePullConfig{}

type StatePullOption interface {
	configureShow(*statePullConfig)
}

func (opt *ReattachOption) configureStatePull(conf *statePullConfig) {
	conf.reattachInfo = opt.info
}

func (tf *Terraform) StatePull(ctx context.Context, opts ...StatePullOption) (string, error) {
	c := defaultStatePullConfig

	for _, o := range opts {
		o.configureShow(&c)
	}

	mergeEnv := map[string]string{}
	if c.reattachInfo != nil {
		reattachStr, err := c.reattachInfo.marshalString()
		if err != nil {
			return "", err
		}
		mergeEnv[reattachEnvVar] = reattachStr
	}

	cmd := tf.statePullCmd(ctx, mergeEnv)

	var ret bytes.Buffer
	cmd.Stdout = &ret
	err := tf.runTerraformCmd(ctx, cmd)
	if err != nil {
		return "", err
	}

	return ret.String(), nil
}

func (tf *Terraform) statePullCmd(ctx context.Context, mergeEnv map[string]string) *exec.Cmd {
	args := []string{"state", "pull"}

	return tf.buildTerraformCmd(ctx, mergeEnv, args...)
}
