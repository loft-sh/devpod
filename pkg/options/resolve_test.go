package options

import (
	"context"
	"testing"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"gotest.tools/assert"
)

type testCase struct {
	Name             string
	ProviderOptions  map[string]*types.Option
	UserValues       map[string]string
	Values           map[string]config.OptionValue
	ExtraValues      map[string]string
	ResolveGlobal    bool
	DontResolveLocal bool
	SkipRequired     bool

	ExpectErr       bool
	ExpectedOptions map[string]string
}

func TestResolveOptions(t *testing.T) {
	testCases := []testCase{
		{
			Name: "simple",
			ExtraValues: map[string]string{
				"WORKSPACE_ID": "test",
			},
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "${WORKSPACE_ID}-test",
				},
			},
			ExpectedOptions: map[string]string{
				"TEST": "test-test",
			},
		},
		{
			Name: "dependency",
			ExtraValues: map[string]string{
				"WORKSPACE_ID": "test",
			},
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "${WORKSPACE_ID}-test-${COMMAND}-$COMMAND",
				},
				"COMMAND": {
					Command: "echo bar",
				},
			},
			ExpectedOptions: map[string]string{
				"TEST":    "test-test-bar-bar",
				"COMMAND": "bar",
			},
		},
		{
			Name: "No extra values",
			ProviderOptions: map[string]*types.Option{
				"COMMAND1": {
					Command: "echo ${COMMAND2}-test",
				},
				"COMMAND2": {
					Command: "echo bar",
				},
			},
			ExpectedOptions: map[string]string{
				"COMMAND1": "bar-test",
				"COMMAND2": "bar",
			},
		},
		{
			Name: "Cyclic dep",
			ProviderOptions: map[string]*types.Option{
				"COMMAND1": {
					Command: "echo ${COMMAND2}",
				},
				"COMMAND2": {
					Command: "echo ${COMMAND1}",
				},
			},
			ExpectErr: true,
		},
		{
			Name: "Override",
			Values: map[string]config.OptionValue{
				"COMMAND": {
					Value:        "foo",
					UserProvided: true,
				},
			},
			ProviderOptions: map[string]*types.Option{
				"COMMAND": {
					Command: "echo bar",
				},
			},
			ExpectedOptions: map[string]string{
				"COMMAND": "foo",
			},
		},
		{
			Name: "Override",
			Values: map[string]config.OptionValue{
				"COMMAND": {
					Value:        "foo",
					UserProvided: true,
				},
			},
			ProviderOptions: map[string]*types.Option{
				"COMMAND": {
					Command: "echo bar",
				},
				"COMMAND1": {
					Command: "echo ${COMMAND}-foo-${UNDEFINED}",
				},
				"DEFAULT1": {
					Default: "${COMMAND}-foo-${UNDEFINED}",
				},
			},
			ExpectedOptions: map[string]string{
				"COMMAND":  "foo",
				"COMMAND1": "foo-foo-",
				"DEFAULT1": "foo-foo-${UNDEFINED}",
			},
		},
		{
			Name: "Expire",
			Values: map[string]config.OptionValue{
				"EXPIRE": {
					Value:  "foo",
					Filled: &[]types.Time{types.NewTime(time.Time{})}[0],
				},
				"NOTEXPIRE": {
					Value:  "foo",
					Filled: &[]types.Time{types.Now()}[0],
				},
			},
			ProviderOptions: map[string]*types.Option{
				"EXPIRE": {
					Command: "echo bar",
					Cache:   "10m",
				},
				"NOTEXPIRE": {
					Command: "echo bar",
					Cache:   "10m",
				},
			},
			ExpectedOptions: map[string]string{
				"EXPIRE":    "bar",
				"NOTEXPIRE": "foo",
			},
		},
		{
			Name: "Ignore self",
			ProviderOptions: map[string]*types.Option{
				"SELF": {
					Command: "SELF=test; echo ${SELF}",
				},
			},
			ExpectedOptions: map[string]string{
				"SELF": "test",
			},
		},
		{
			Name: "Recompute children",
			UserValues: map[string]string{
				"PARENT": "foo",
			},
			Values: map[string]config.OptionValue{
				"PARENT": {
					Value:        "test",
					UserProvided: true,
				},
				"CHILD1": {
					Value: "test-child1",
				},
				"CHILD2": {
					Value: "test-child2",
				},
			},
			ProviderOptions: map[string]*types.Option{
				"PARENT": {},
				"CHILD1": {
					Command: "echo ${PARENT}-child1",
				},
				"CHILD2": {
					Default: "${PARENT}-child2",
				},
			},
			ExpectedOptions: map[string]string{
				"PARENT": "foo",
				"CHILD1": "foo-child1",
				"CHILD2": "foo-child2",
			},
		},
		{
			Name: "Error local global",
			ProviderOptions: map[string]*types.Option{
				"PARENT": {
					Default: "test",
				},
				"CHILD1": {
					Global:  true,
					Default: "${PARENT}",
				},
			},
			ExpectErr: true,
		},
		{
			Name: "Error local var",
			ProviderOptions: map[string]*types.Option{
				"PARENT": {
					Local:   true,
					Default: "test",
				},
				"CHILD1": {
					Default: "${PARENT}",
				},
			},
			ExpectErr: true,
		},
		{
			Name: "Don't resolve local",
			ProviderOptions: map[string]*types.Option{
				"PARENT": {
					Default: "test",
				},
				"CHILD1": {
					Default: "${PARENT}",
					Local:   true,
				},
			},
			DontResolveLocal: true,
			ExpectedOptions: map[string]string{
				"PARENT": "test",
			},
		},
		{
			Name: "Resolve",
			ProviderOptions: map[string]*types.Option{
				"PARENT": {
					Default: "test",
				},
				"CHILD1": {
					Default: "${PARENT}",
				},
			},
			DontResolveLocal: true,
			ExpectedOptions: map[string]string{
				"PARENT": "test",
				"CHILD1": "test",
			},
		},
		{
			Name: "Skip Required",
			ProviderOptions: map[string]*types.Option{
				"PARENT": {
					Required: true,
				},
				"CHILD1": {
					Default: "${PARENT}",
				},
				"PARENT2": {
					Required: true,
					Default:  "test",
				},
				"CHILD2": {
					Default: "${PARENT2}",
				},
			},
			SkipRequired: true,
			ExpectedOptions: map[string]string{
				"PARENT2": "test",
				"CHILD2":  "test",
			},
		},
	}

	for _, testCase := range testCases {
		options, _, err := resolveOptionsGeneric(context.Background(), testCase.ProviderOptions, testCase.Values, testCase.UserValues, testCase.ExtraValues, !testCase.DontResolveLocal, testCase.ResolveGlobal, testCase.SkipRequired, log.Default)
		if !testCase.ExpectErr {
			assert.NilError(t, err, testCase.Name)
		} else if testCase.ExpectErr {
			if err == nil {
				t.Fatalf("expected error, got nil error in test case %s", testCase.Name)
			}

			continue
		}

		strOptions := map[string]string{}
		for k, v := range options {
			strOptions[k] = v.Value
		}
		if len(testCase.ExpectedOptions) > 0 {
			assert.DeepEqual(t, strOptions, testCase.ExpectedOptions)
		} else {
			assert.DeepEqual(t, strOptions, map[string]string{})
		}
	}
}
