package tfexec

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type graphConfig struct {
	plan       string
	drawCycles bool
	graphType  string
}

var defaultGraphOptions = graphConfig{}

type GraphOption interface {
	configureGraph(*graphConfig)
}

func (opt *GraphPlanOption) configureGraph(conf *graphConfig) {
	conf.plan = opt.file
}

func (opt *DrawCyclesOption) configureGraph(conf *graphConfig) {
	conf.drawCycles = opt.drawCycles
}

func (opt *GraphTypeOption) configureGraph(conf *graphConfig) {
	conf.graphType = opt.graphType
}

func (tf *Terraform) Graph(ctx context.Context, opts ...GraphOption) (string, error) {
	graphCmd, err := tf.graphCmd(ctx, opts...)
	if err != nil {
		return "", err
	}
	var outBuf strings.Builder
	graphCmd.Stdout = &outBuf
	err = tf.runTerraformCmd(ctx, graphCmd)
	if err != nil {
		return "", err
	}

	return outBuf.String(), nil

}

func (tf *Terraform) graphCmd(ctx context.Context, opts ...GraphOption) (*exec.Cmd, error) {
	c := defaultGraphOptions

	for _, o := range opts {
		o.configureGraph(&c)
	}

	args := []string{"graph"}

	if c.plan != "" {
		// plan was a positional arguement prior to Terraform 0.15.0. Ensure proper use by checking version.
		if err := tf.compatible(ctx, tf0_15_0, nil); err == nil {
			args = append(args, "-plan="+c.plan)
		} else {
			args = append(args, c.plan)
		}
	}

	if c.drawCycles {
		err := tf.compatible(ctx, tf0_5_0, nil)
		if err != nil {
			return nil, fmt.Errorf("-draw-cycles was first introduced in Terraform 0.5.0: %w", err)
		}
		args = append(args, "-draw-cycles")
	}

	if c.graphType != "" {
		err := tf.compatible(ctx, tf0_8_0, nil)
		if err != nil {
			return nil, fmt.Errorf("-graph-type was first introduced in Terraform 0.8.0: %w", err)
		}
		args = append(args, "-type="+c.graphType)
	}

	return tf.buildTerraformCmd(ctx, nil, args...), nil
}
