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
	Name             string
	ProviderOptions  map[string]*types.Option
	UserValues       map[string]string
	ResolvedValues   map[string]config.OptionValue
	ExtraValues      map[string]string
	ResolveGlobal    bool
	DontResolveLocal bool
	SkipRequired     bool

	ExpectErr              bool
	ExpectedOptions        map[string]string
	ExpectedDynamicOptions config.DynamicOptions
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
			Name: "Simple dynamic options (unresolved children)",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default:           "test",
					SubOptionsCommand: `echo '{ "options": { "FOO": { "command": "echo bar" } } }'`,
				},
			},
			ExpectedOptions:        map[string]string{"TEST": "test"},
			ExpectedDynamicOptions: config.DynamicOptions{"FOO": &types.Option{Command: "echo bar"}},
		},
		{
			Name: "Dynamic option with resolved parent",
			ProviderOptions: map[string]*types.Option{
				"TEST": {
					Default:           "test",
					SubOptionsCommand: `echo '{ "options": { "FOO": { "command": "echo bar" } } }'`,
				},
				"FOO": {Command: "echo bar"},
			},
			ResolvedValues: map[string]config.OptionValue{
				"TEST": {Value: "test", Children: []string{"FOO"}},
			},
			ExpectedOptions:        map[string]string{"TEST": "test", "FOO": "bar"},
			ExpectedDynamicOptions: config.DynamicOptions{},
		},
	}

	for _, testCase := range testCases {
		resolverOpts := []resolver.Option{resolver.WithSkipRequired(testCase.SkipRequired)}
		if !testCase.DontResolveLocal {
			resolverOpts = append(resolverOpts, resolver.WithResolveLocal())
		}
		if testCase.ResolveGlobal {
			resolverOpts = append(resolverOpts, resolver.WithResolveGlobal())
		}
		r := resolver.New(testCase.UserValues, testCase.ExtraValues, log.Default, resolverOpts...)
		options, dynamicOptions, err := r.Resolve(context.Background(), testCase.ProviderOptions, testCase.ResolvedValues)
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
			assert.DeepEqual(t, dynamicOptions, config.DynamicOptions{})
		}
	}
}

type TestCaseDevConfig struct {
	Name           string
	DevConfig      *config.Config
	ProviderConfig *provider.ProviderConfig
	Init           bool
	SkipRequired   bool
	UserValues     map[string]string

	ExpectErr         bool
	ExpectedDevConfig *config.Config
}

func TestResolveOptionsDevConfig(t *testing.T) {
	singleMachine := false
	binaries := map[string][]*provider.ProviderBinary{}
	providerName := "test-provider"
	testCases := []TestCaseDevConfig{}

	// // No options
	// withoutOptionsProviderConfig := provider.ProviderConfig{Name: providerName, Binaries: binaries}
	// testCases = append(testCases, TestCaseDevConfig{
	// 	Name:              "No options",
	// 	UserValues:        map[string]string{},
	// 	DevConfig:         initDevConfig(&config.Config{}, withoutOptionsProviderConfig),
	// 	ProviderConfig:    &withoutOptionsProviderConfig,
	// 	Init:              false,
	// 	ExpectedDevConfig: initDevConfig(&config.Config{}, withoutOptionsProviderConfig)},
	// )
	//
	// // Simple with SubOptions (init)
	// simpleSubOptionsProviderConfig := provider.ProviderConfig{
	// 	Name:     providerName,
	// 	Binaries: binaries,
	// 	Options:  map[string]*provider.ProviderOption{"TEST": {Option: types.Option{Default: "test", SubOptionsCommand: `echo '{ "options": { "BAR": { "command": "echo bar" } } }'`}}},
	// }
	// expectedDevConfig := initDevConfig(&config.Config{}, simpleSubOptionsProviderConfig)
	// expectedDevConfig.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR"}}, "BAR": {Value: "bar"}}
	// expectedDevConfig.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Command: "echo bar"}}
	// testCases = append(testCases, TestCaseDevConfig{
	// 	Name:              "Simple with SubOptions (init)",
	// 	UserValues:        map[string]string{},
	// 	DevConfig:         initDevConfig(&config.Config{}, simpleSubOptionsProviderConfig),
	// 	ProviderConfig:    &simpleSubOptionsProviderConfig,
	// 	Init:              true,
	// 	ExpectedDevConfig: expectedDevConfig},
	// )
	//
	// // Simple with SubOptions (no init)
	// expectedDevConfig2 := initDevConfig(&config.Config{}, simpleSubOptionsProviderConfig)
	// expectedDevConfig2.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR"}}}
	// expectedDevConfig2.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Command: "echo bar"}}
	// testCases = append(testCases, TestCaseDevConfig{
	// 	Name:              "Simple with SubOptions (no init)",
	// 	UserValues:        map[string]string{},
	// 	DevConfig:         initDevConfig(&config.Config{}, simpleSubOptionsProviderConfig),
	// 	ProviderConfig:    &simpleSubOptionsProviderConfig,
	// 	Init:              false,
	// 	ExpectedDevConfig: expectedDevConfig2},
	// )
	//
	// // SubOption default value (init)
	// subOptionDefaultValueProviderConfig := provider.ProviderConfig{
	// 	Name:     providerName,
	// 	Binaries: binaries,
	// 	Options:  map[string]*provider.ProviderOption{"TEST": {Option: types.Option{Default: "test", SubOptionsCommand: `echo '{ "options": { "BAR": { "command": "echo bar" }, "BAZ": { "default": "baz" } } }'`}}},
	// }
	// expectedDevConfig3 := initDevConfig(&config.Config{}, subOptionDefaultValueProviderConfig)
	// expectedDevConfig3.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR", "BAZ"}}, "BAR": {Value: "bar"}, "BAZ": {Value: "baz"}}
	// expectedDevConfig3.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Command: "echo bar"}, "BAZ": {Default: "baz"}}
	// testCases = append(testCases, TestCaseDevConfig{
	// 	Name:              "SubOptions default value (init)",
	// 	UserValues:        map[string]string{},
	// 	DevConfig:         initDevConfig(&config.Config{}, subOptionDefaultValueProviderConfig),
	// 	ProviderConfig:    &subOptionDefaultValueProviderConfig,
	// 	Init:              true,
	// 	ExpectedDevConfig: expectedDevConfig3},
	// )
	//
	// // SubOption default value (no init)
	// expectedDevConfig4 := initDevConfig(&config.Config{}, subOptionDefaultValueProviderConfig)
	// expectedDevConfig4.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR", "BAZ"}}}
	// expectedDevConfig4.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Command: "echo bar"}, "BAZ": {Default: "baz"}}
	// testCases = append(testCases, TestCaseDevConfig{
	// 	Name:              "SubOptions default value (no init)",
	// 	UserValues:        map[string]string{},
	// 	DevConfig:         initDevConfig(&config.Config{}, subOptionDefaultValueProviderConfig),
	// 	ProviderConfig:    &subOptionDefaultValueProviderConfig,
	// 	Init:              false,
	// 	ExpectedDevConfig: expectedDevConfig4},
	// )
	//
	// // SubOption required (init)
	// subOptionRequiredProviderConfig := provider.ProviderConfig{
	// 	Name:     providerName,
	// 	Binaries: binaries,
	// 	Options:  map[string]*provider.ProviderOption{"TEST": {Option: types.Option{Default: "test", SubOptionsCommand: `echo '{ "options": { "BAR": { "required": true, "command": "echo bar" }, "BAZ": { "required": false } } }'`}}},
	// }
	// expectedDevConfig5 := initDevConfig(&config.Config{}, subOptionRequiredProviderConfig)
	// expectedDevConfig5.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR", "BAZ"}}, "BAR": {Value: "bar"}, "BAZ": {Value: ""}}
	// expectedDevConfig5.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Required: true, Command: "echo bar"}, "BAZ": {Required: false}}
	// testCases = append(testCases, TestCaseDevConfig{
	// 	Name:              "SubOptions default value (init)",
	// 	UserValues:        map[string]string{},
	// 	DevConfig:         initDevConfig(&config.Config{}, subOptionRequiredProviderConfig),
	// 	ProviderConfig:    &subOptionRequiredProviderConfig,
	// 	Init:              true,
	// 	ExpectedDevConfig: expectedDevConfig5},
	// )
	//
	// // SubOptions cylcic dependency (init)
	// subOptionCircularProviderConfig := provider.ProviderConfig{
	// 	Name:     providerName,
	// 	Binaries: binaries,
	// 	Options: map[string]*provider.ProviderOption{
	// 		"TEST":  {Option: types.Option{Default: "test", SubOptionsCommand: `echo '{ "options": { "BAR": { "required": true, "command": "echo ${TEST2}" } } }'`}},
	// 		"TEST2": {Option: types.Option{Default: "test2", Command: "echo ${BAR}"}},
	// 	}}
	// expectedDevConfig6 := initDevConfig(&config.Config{}, subOptionCircularProviderConfig)
	// expectedDevConfig6.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR", "BAZ"}}, "BAR": {Value: "bar"}, "BAZ": {Value: ""}}
	// expectedDevConfig6.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Required: true, Command: "echo bar"}, "BAZ": {Required: false}}
	// testCases = append(testCases, TestCaseDevConfig{
	// 	Name:           "SubOptions cylcic dependency (init)",
	// 	UserValues:     map[string]string{},
	// 	DevConfig:      initDevConfig(&config.Config{}, subOptionCircularProviderConfig),
	// 	ProviderConfig: &subOptionCircularProviderConfig,
	// 	Init:           true,
	// 	ExpectErr:      true,
	// },
	// )

	// Nested SubOptions (init)
	subOpts1 := provider.SubOptions{Options: map[string]types.Option{
		"BAZ": {Command: "echo baz"},
	}}
	subOpts1Bytes, err := json.Marshal(subOpts1)
	if err != nil {
		t.Fatal("Nested SubOptions", err)
	}
	s1 := base64.StdEncoding.EncodeToString(subOpts1Bytes)
	barSubOptionsCommand := fmt.Sprintf("echo %s | base64 -d", s1)
	subOpts2 := provider.SubOptions{Options: map[string]types.Option{
		"BAR": {Command: "echo bar", SubOptionsCommand: barSubOptionsCommand},
	}}
	subOpts2Bytes, err := json.Marshal(subOpts2)
	if err != nil {
		t.Fatal("Nested SubOptions", err)
	}
	s2 := base64.StdEncoding.EncodeToString(subOpts2Bytes)
	subOpts3Str := fmt.Sprintf("echo %s | base64 -d", s2)
	subOptionNestedProviderConfig := provider.ProviderConfig{
		Name:     providerName,
		Binaries: binaries,
		Options: map[string]*provider.ProviderOption{
			"TEST": {Option: types.Option{Default: "test", SubOptionsCommand: subOpts3Str}},
		}}
	expectedDevConfig7 := initDevConfig(&config.Config{}, subOptionNestedProviderConfig)
	expectedDevConfig7.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR"}}, "BAR": {Value: "bar", Children: []string{"BAZ"}}, "BAZ": {Value: "baz"}}
	expectedDevConfig7.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Command: "echo bar", SubOptionsCommand: barSubOptionsCommand}, "BAZ": {Command: "echo baz"}}
	testCases = append(testCases, TestCaseDevConfig{
		Name:              "Nested SubOptions (init)",
		UserValues:        map[string]string{},
		DevConfig:         initDevConfig(&config.Config{}, subOptionNestedProviderConfig),
		ProviderConfig:    &subOptionNestedProviderConfig,
		Init:              true,
		ExpectedDevConfig: expectedDevConfig7,
	},
	)

	for _, testCase := range testCases {
		newConfig, err := ResolveOptions(context.Background(), testCase.DevConfig, testCase.ProviderConfig, testCase.UserValues, testCase.SkipRequired, &singleMachine, testCase.Init, log.Default)
		if !testCase.ExpectErr {
			assert.NilError(t, err, testCase.Name)
		} else if testCase.ExpectErr {
			if err == nil {
				t.Fatalf("expected error, got nil error in test case %s", testCase.Name)
			}

			continue
		}

		assertOptions(t, newConfig, testCase.ExpectedDevConfig, providerName)
	}
}

func TestSubOptionCyclicDependencyNoInit(t *testing.T) {
	singleMachine := false
	binaries := map[string][]*provider.ProviderBinary{}
	providerName := "test-provider"

	providerConfig := provider.ProviderConfig{
		Name:     providerName,
		Binaries: binaries,
		Options: map[string]*provider.ProviderOption{
			"TEST":  {Option: types.Option{Default: "test", SubOptionsCommand: `echo '{ "options": { "BAR": { "required": true, "command": "echo ${BAZ}" } } }'`}},
			"TEST2": {Option: types.Option{Default: "test2", SubOptionsCommand: `echo '{ "options": { "BAZ": { "required": true, "command": "echo ${BAR}" } } }'`}},
		},
	}
	devConfig := initDevConfig(&config.Config{}, providerConfig)
	devConfig.Current().Providers[providerName].Options = map[string]config.OptionValue{"TEST": {Value: "test", Children: []string{"BAR"}}, "TEST2": {Value: "test2", Children: []string{"BAZ"}}}
	devConfig.Current().Providers[providerName].DynamicOptions = config.DynamicOptions{"BAR": {Required: true, Command: "echo ${BAZ}"}, "BAZ": {Required: true, Command: "echo ${BAR}"}}
	// First tick should not error as we can't detect cyclic dependencies without the dynamic options yet
	newConfig, err := ResolveOptions(context.Background(), initDevConfig(&config.Config{}, providerConfig), &providerConfig, map[string]string{}, false, &singleMachine, false, log.Default)
	assert.NilError(t, err)
	assertOptions(t, newConfig, devConfig, providerName)

	// now we have the dynamic options, we should detect the cyclic dependency
	_, err = ResolveOptions(context.Background(), newConfig, &providerConfig, map[string]string{}, false, &singleMachine, true, log.Default)
	assert.ErrorContains(t, err, "cyclic provider option")
}

func assertOptions(t *testing.T, newConfig *config.Config, expectedConfig *config.Config, providerName string) {
	// options
	gotOpts := map[string]string{}
	gotOptsChildren := map[string][]string{}
	for k, v := range newConfig.Current().Providers[providerName].Options {
		gotOpts[k] = v.Value
		gotOptsChildren[k] = v.Children
	}
	wantOpts := map[string]string{}
	wantOptsChildren := map[string][]string{}
	for k, v := range expectedConfig.Current().Providers[providerName].Options {
		wantOpts[k] = v.Value
		wantOptsChildren[k] = v.Children
	}

	assert.DeepEqual(t, gotOpts, wantOpts)
	assert.DeepEqual(t, gotOptsChildren, wantOptsChildren)

	// dynamic options
	gotDynamicOpts := config.DynamicOptions{}
	for k, v := range newConfig.Current().Providers[providerName].DynamicOptions {
		gotDynamicOpts[k] = v
	}
	wantDynamicOpts := config.DynamicOptions{}
	for k, v := range expectedConfig.Current().Providers[providerName].DynamicOptions {
		wantDynamicOpts[k] = v
	}

	assert.DeepEqual(t, gotDynamicOpts, wantDynamicOpts)
}

func initDevConfig(devConfig *config.Config, provider provider.ProviderConfig) *config.Config {
	if devConfig.DefaultContext == "" {
		devConfig.DefaultContext = "default"
	}
	if devConfig.Contexts == nil {
		devConfig.Contexts = map[string]*config.ContextConfig{
			"default": {
				DefaultProvider: provider.Name,
				Providers:       map[string]*config.ProviderConfig{},
				IDEs:            map[string]*config.IDEConfig{},
				Options:         nil,
			},
		}
	}
	if devConfig.Current().Providers == nil {
		devConfig.Current().Providers = map[string]*config.ProviderConfig{}
	}
	if devConfig.Current().Providers[provider.Name] == nil {
		devConfig.Current().Providers[provider.Name] = &config.ProviderConfig{}
	}
	devConfig.Current().Providers[provider.Name].Options = map[string]config.OptionValue{}
	devConfig.Current().Providers[provider.Name].DynamicOptions = config.DynamicOptions{}

	return devConfig
}
