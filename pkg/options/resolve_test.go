package options

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/options/resolver"
	"github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log"
	"gotest.tools/assert"
)

type testCase struct {
	Name                       string
	ProviderOptions            map[string]*types.Option
	UserValues                 map[string]string
	ResolvedValues             map[string]config.OptionValue
	ResolvedDynamicDefinitions config.OptionDefinitions
	ExtraValues                map[string]string
	ResolveGlobal              bool
	DontResolveLocal           bool
	SkipRequired               bool

	ExpectErr              bool
	ExpectedOptions        map[string]string
	ExpectedDynamicOptions config.OptionDefinitions
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
			ResolvedValues: map[string]config.OptionValue{
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
			ResolvedValues: map[string]config.OptionValue{
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
			ResolvedValues: map[string]config.OptionValue{
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
			ResolvedValues: map[string]config.OptionValue{
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
		{
			Name: "Nested dynamic options",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Default: "test2",
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test",
				"TEST2": "test2",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": &types.Option{
					Default: "test2",
				},
			},
		},
		{
			Name: "Dynamic options don't update",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Default: "test2",
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			ResolvedDynamicDefinitions: map[string]*types.Option{
				"TEST2": {
					Default: "test5",
				},
			},
			ResolvedValues: map[string]config.OptionValue{
				"TEST":  {Value: "test3", Children: []string{"TEST2"}, UserProvided: true},
				"TEST2": {Value: "test4", UserProvided: true},
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test3",
				"TEST2": "test4",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": {
					Default: "test2",
				},
			},
		},
		{
			Name: "Dynamic options update",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST3": &types.Option{
							Default: "test2",
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			UserValues: map[string]string{
				"TEST": "test1",
			},
			ResolvedValues: map[string]config.OptionValue{
				"TEST":  {Value: "test3", Children: []string{"TEST2"}},
				"TEST2": {Value: "test4"},
			},
			ResolvedDynamicDefinitions: map[string]*types.Option{
				"TEST2": {
					Default: "test5",
				},
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test1",
				"TEST3": "test2",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST3": &types.Option{
					Default: "test2",
				},
			},
		},
		{
			Name: "Nested dynamic options",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test1",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Default: "test2",
							SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
								"TEST3": &types.Option{
									Default: "test3",
									SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
										"TEST4": &types.Option{
											Default: "${TEST3}-${FOO}-4",
										},
									}),
								},
							}),
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test1",
				"TEST2": "test2",
				"TEST3": "test3",
				"TEST4": "test3-bar-4",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": &types.Option{
					Default: "test2",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST3": &types.Option{
							Default: "test3",
							SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
								"TEST4": &types.Option{
									Default: "${TEST3}-${FOO}-4",
								},
							}),
						},
					}),
				},
				"TEST3": &types.Option{
					Default: "test3",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST4": &types.Option{
							Default: "${TEST3}-${FOO}-4",
						},
					}),
				},
				"TEST4": &types.Option{
					Default: "${TEST3}-${FOO}-4",
				},
			},
		},
		{
			Name: "Nested dynamic options skip required",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test1",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Required: true,
							SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
								"TEST3": &types.Option{
									Default: "test3",
									SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
										"TEST4": &types.Option{
											Default: "${TEST3}-${FOO}-4",
										},
									}),
								},
							}),
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			SkipRequired: true,
			ExpectedOptions: map[string]string{
				"TEST": "test1",
				"FOO":  "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": &types.Option{
					Required: true,
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST3": &types.Option{
							Default: "test3",
							SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
								"TEST4": &types.Option{
									Default: "${TEST3}-${FOO}-4",
								},
							}),
						},
					}),
				},
			},
		},
		{
			Name: "Nested dynamic options use option",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test1",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Required: true,
							SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
								"TEST3": &types.Option{
									Default: "test3",
									SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
										"TEST4": &types.Option{
											Default: "${TEST2}-${FOO}-4",
										},
									}),
								},
							}),
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			SkipRequired: true,
			UserValues: map[string]string{
				"TEST2": "test2",
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test1",
				"TEST2": "test2",
				"TEST3": "test3",
				"TEST4": "test2-bar-4",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": &types.Option{
					Required: true,
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST3": &types.Option{
							Default: "test3",
							SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
								"TEST4": &types.Option{
									Default: "${TEST2}-${FOO}-4",
								},
							}),
						},
					}),
				},
				"TEST3": &types.Option{
					Default: "test3",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST4": &types.Option{
							Default: "${TEST2}-${FOO}-4",
						},
					}),
				},
				"TEST4": &types.Option{
					Default: "${TEST2}-${FOO}-4",
				},
			},
		},
		{
			Name: "Nested dynamic options use option",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test1",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Default: "test2",
							SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
								"TEST3": &types.Option{
									Default: "test3",
								},
							}),
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			ResolvedValues: map[string]config.OptionValue{
				"TEST5": {
					Value: "test5",
				},
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test1",
				"TEST2": "test2",
				"TEST3": "test3",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": &types.Option{
					Default: "test2",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST3": &types.Option{
							Default: "test3",
						},
					}),
				},
				"TEST3": &types.Option{
					Default: "test3",
				},
			},
		},
		{
			Name: "Dynamic options unused option",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test1",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Default: "test2",
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			ResolvedValues: map[string]config.OptionValue{
				"TEST5": {
					Value: "test5",
				},
			},
			ResolvedDynamicDefinitions: map[string]*types.Option{
				"TEST5": &types.Option{
					Default: "test2",
				},
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test1",
				"TEST2": "test2",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": &types.Option{
					Default: "test2",
				},
			},
		},
		{
			Name: "Dynamic options update default",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default: "test1",
					SubOptionsCommand: optionsToSubCommand(config.OptionDefinitions{
						"TEST2": &types.Option{
							Default: "test3",
						},
					}),
				},
				"FOO": {Command: "echo bar"},
			},
			ResolvedValues: map[string]config.OptionValue{
				"TEST": {
					Value: "test1",
				},
				"TEST2": {
					Value: "test2",
				},
			},
			ResolvedDynamicDefinitions: map[string]*types.Option{
				"TEST2": {
					Default: "test2",
				},
			},
			ExpectedOptions: map[string]string{
				"TEST":  "test1",
				"TEST2": "test3",
				"FOO":   "bar",
			},
			ExpectedDynamicOptions: config.OptionDefinitions{
				"TEST2": &types.Option{
					Default: "test3",
				},
			},
		},
	}

	for _, testCase := range testCases {
		fmt.Println(testCase.Name)
		resolverOpts := []resolver.Option{resolver.WithSkipRequired(testCase.SkipRequired), resolver.WithResolveSubOptions()}
		if !testCase.DontResolveLocal {
			resolverOpts = append(resolverOpts, resolver.WithResolveLocal())
		}
		if testCase.ResolveGlobal {
			resolverOpts = append(resolverOpts, resolver.WithResolveGlobal())
		}
		r := resolver.New(testCase.UserValues, testCase.ExtraValues, log.Default, resolverOpts...)
		options, dynamicOptions, err := r.Resolve(context.Background(), testCase.ResolvedDynamicDefinitions, testCase.ProviderOptions, testCase.ResolvedValues)
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

		if len(testCase.ExpectedDynamicOptions) > 0 {
			assert.DeepEqual(t, dynamicOptions, testCase.ExpectedDynamicOptions)
		} else {
			assert.DeepEqual(t, dynamicOptions, config.OptionDefinitions{})
		}
	}
}

func optionsToSubCommand(optionDefinitions config.OptionDefinitions) string {
	out, _ := json.Marshal(&provider.SubOptions{
		Options: optionDefinitions,
	})
	return fmt.Sprintf("echo '%s' | base64 --decode", base64.StdEncoding.EncodeToString(out))
}
