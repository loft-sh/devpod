package options

import (
	"context"
	provider2 "github.com/loft-sh/devpod/pkg/provider"
	"github.com/loft-sh/devpod/pkg/types"
	"gotest.tools/assert"
	"reflect"
	"testing"
	"time"
)

type testCase struct {
	Name            string
	BeforeStage     string
	AfterStage      string
	Workspace       *provider2.Workspace
	ProviderOptions map[string]*provider2.ProviderOption

	ExpectNotChanged bool
	ExpectErr        bool
	ExpectedOptions  map[string]string
}

func TestResolveOptions(t *testing.T) {
	testCases := []testCase{
		{
			Name:      "simple",
			Workspace: &provider2.Workspace{ID: "test"},
			ProviderOptions: map[string]*provider2.ProviderOption{
				"TEST": {
					Default: "${WORKSPACE_ID}-test",
				},
			},
			ExpectedOptions: map[string]string{
				"TEST": "test-test",
			},
		},
		{
			Name:      "dependency",
			Workspace: &provider2.Workspace{ID: "test"},
			ProviderOptions: map[string]*provider2.ProviderOption{
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
			Name: "Nil workspace",
			ProviderOptions: map[string]*provider2.ProviderOption{
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
			ProviderOptions: map[string]*provider2.ProviderOption{
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
			Name:       "Later stage",
			AfterStage: "command",
			ProviderOptions: map[string]*provider2.ProviderOption{
				"COMMAND1": {
					Command: "echo ${COMMAND2}",
					After:   "command",
				},
				"COMMAND2": {
					Command: "echo bar",
				},
			},
			ExpectErr: true,
		},
		{
			Name:       "Correct stage",
			AfterStage: "command",
			Workspace: &provider2.Workspace{
				ID: "test",
				Provider: provider2.WorkspaceProviderConfig{
					Options: map[string]provider2.OptionValue{
						"COMMAND2": {
							Value:   "bar",
							Expires: &[]types.Time{types.NewTime(time.Time{})}[0],
						},
					},
				},
			},
			ProviderOptions: map[string]*provider2.ProviderOption{
				"COMMAND1": {
					Command: "echo ${COMMAND2}",
					After:   "command",
				},
				"COMMAND2": {
					Command: "echo bar",
				},
			},
			ExpectedOptions: map[string]string{
				"COMMAND1": "bar",
				"COMMAND2": "bar",
			},
		},
		{
			Name: "Override",
			Workspace: &provider2.Workspace{
				ID: "test",
				Provider: provider2.WorkspaceProviderConfig{
					Options: map[string]provider2.OptionValue{
						"COMMAND": {
							Value: "foo",
						},
					},
				},
			},
			ProviderOptions: map[string]*provider2.ProviderOption{
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
			Workspace: &provider2.Workspace{
				ID: "test",
				Provider: provider2.WorkspaceProviderConfig{
					Options: map[string]provider2.OptionValue{
						"COMMAND": {
							Value: "foo",
						},
					},
				},
			},
			ProviderOptions: map[string]*provider2.ProviderOption{
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
			Workspace: &provider2.Workspace{
				ID: "test",
				Provider: provider2.WorkspaceProviderConfig{
					Options: map[string]provider2.OptionValue{
						"EXPIRE": {
							Value:   "foo",
							Expires: &[]types.Time{types.NewTime(time.Time{})}[0],
						},
						"NOTEXPIRE": {
							Value:   "foo",
							Expires: &[]types.Time{types.NewTime(time.Now().Add(time.Hour))}[0],
						},
					},
				},
			},
			ProviderOptions: map[string]*provider2.ProviderOption{
				"EXPIRE": {
					Command: "echo bar",
				},
				"NOTEXPIRE": {
					Command: "echo bar",
				},
			},
			ExpectedOptions: map[string]string{
				"EXPIRE":    "bar",
				"NOTEXPIRE": "foo",
			},
		},
		{
			Name:        "Expire Stage",
			BeforeStage: "init",
			Workspace: &provider2.Workspace{
				ID: "test",
				Provider: provider2.WorkspaceProviderConfig{
					Options: map[string]provider2.OptionValue{
						"EXPIRE": {
							Value:   "foo",
							Expires: &[]types.Time{types.NewTime(time.Time{})}[0],
						},
						"NOTEXPIRE": {
							Value:   "foo",
							Expires: &[]types.Time{types.NewTime(time.Now().Add(time.Hour))}[0],
						},
					},
				},
			},
			ProviderOptions: map[string]*provider2.ProviderOption{
				"EXPIRE": {
					Command: "echo bar",
				},
				"NOTEXPIRE": {
					Command: "echo bar",
				},
			},
			ExpectedOptions: map[string]string{
				"EXPIRE":    "foo",
				"NOTEXPIRE": "foo",
			},
		},
		{
			Name:        "No change",
			BeforeStage: "init",
			Workspace: &provider2.Workspace{
				ID: "test",
				Provider: provider2.WorkspaceProviderConfig{
					Options: map[string]provider2.OptionValue{
						"EXPIRE": {
							Value:   "foo",
							Expires: &[]types.Time{types.NewTime(time.Time{})}[0],
						},
						"NOTEXPIRE": {
							Value:   "foo",
							Expires: &[]types.Time{types.NewTime(time.Now().Add(time.Hour))}[0],
						},
					},
				},
			},
			ProviderOptions: map[string]*provider2.ProviderOption{
				"EXPIRE": {
					Command: "echo bar",
				},
				"NOTEXPIRE": {
					Command: "echo bar",
				},
			},
			ExpectNotChanged: true,
		},
		{
			Name: "Ignore self",
			ProviderOptions: map[string]*provider2.ProviderOption{
				"SELF": {
					Command: "SELF=test; echo ${SELF}",
				},
			},
			ExpectedOptions: map[string]string{
				"SELF": "test",
			},
		},
	}

	for _, testCase := range testCases {
		options, err := ResolveOptions(context.Background(), testCase.BeforeStage, testCase.AfterStage, testCase.Workspace, testCase.ProviderOptions)
		if !testCase.ExpectErr {
			assert.NilError(t, err, testCase.Name)
		} else if testCase.ExpectErr {
			if err == nil {
				t.Fatalf("expected error, got nil error in test case %s", testCase.Name)
			}

			continue
		}

		if len(testCase.ExpectedOptions) > 0 {
			strOptions := map[string]string{}
			for k, v := range options {
				strOptions[k] = v.Value
			}

			assert.DeepEqual(t, strOptions, testCase.ExpectedOptions)
		}

		if testCase.ExpectNotChanged {
			assert.DeepEqual(t, options, testCase.Workspace.Provider.Options)
			assert.DeepEqual(t, reflect.DeepEqual(options, testCase.Workspace.Provider.Options), true)
		}
	}
}
