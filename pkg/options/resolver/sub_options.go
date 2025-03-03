package resolver

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/shell"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/pkg/errors"
)

func execOptionCommand(ctx context.Context, command string, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (*bytes.Buffer, error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := os.Environ()
	for k, v := range combine(resolvedOptions, extraValues) {
		env = append(env, k+"="+v)
	}

	err := shell.RunEmulatedShell(ctx, command, nil, stdout, stderr, env)
	if err != nil {
		return nil, errors.Wrapf(err, "exec command: %s%s", stdout.String(), stderr.String())
	}

	return stdout, nil
}

func resolveFromCommand(ctx context.Context, option *types.Option, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (config.OptionValue, error) {
	cmdOut, err := execOptionCommand(ctx, option.Command, resolvedOptions, extraValues)
	if err != nil {
		return config.OptionValue{}, errors.Wrap(err, "run command")
	}
	optionValue := config.OptionValue{Value: strings.TrimSpace(cmdOut.String())}
	expire := types.NewTime(time.Now())
	optionValue.Filled = &expire
	return optionValue, nil
}

func resolveSubOptions(ctx context.Context, option *types.Option, resolvedOptions map[string]config.OptionValue, extraValues map[string]string) (config.OptionDefinitions, error) {
	cmdOut, err := execOptionCommand(ctx, option.SubOptionsCommand, resolvedOptions, extraValues)
	if err != nil {
		return nil, errors.Wrap(err, "run subOptionsCommand")
	}
	subOptions := provider.SubOptions{}
	err = yaml.Unmarshal(cmdOut.Bytes(), &subOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "parse subOptionsCommand: %s", cmdOut.String())
	}

	// prepare new options
	// need to look for option in graph. should be rather easy because we don't need to traverse the whole graph
	retOpts := config.OptionDefinitions{}
	for k, v := range subOptions.Options {
		retOpts[k] = v
	}

	return retOpts, nil
}
