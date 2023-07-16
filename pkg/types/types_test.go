package types_test

import (
	"encoding/json"
	"testing"

	"github.com/loft-sh/devpod/pkg/types"
	"gotest.tools/assert"
)

func TestLifecycleHookUnmarshalJSON(t *testing.T) {
	type input struct {
		Input types.LifecycleHook `json:"input,omitempty"`
	}

	testCases := []struct {
		Name   string
		Input  string
		Expect input
	}{
		{
			Name:  "string",
			Input: `{"input": "some-string"}`,
			Expect: input{
				Input: types.LifecycleHook{
					"": []string{"some-string"},
				},
			},
		},
		{
			Name:  "array of strings",
			Input: `{"input": ["string1", "string2"]}`,
			Expect: input{
				Input: types.LifecycleHook{
					"": []string{
						"string1",
						"string2",
					},
				},
			},
		},
		{
			Name:  "object of strings",
			Input: `{"input": {"key1": "value1", "key2": "value2"}}`,
			Expect: input{
				Input: types.LifecycleHook{
					"key1": []string{
						"value1",
					},
					"key2": []string{
						"value2",
					},
				},
			},
		},
		{
			Name:  "object of array of strings",
			Input: `{"input": {"key1": ["value1","value2"], "key2": ["value3","value4"]}}`,
			Expect: input{
				Input: types.LifecycleHook{
					"key1": []string{
						"value1",
						"value2",
					},
					"key2": []string{
						"value3",
						"value4",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			var data input

			err := json.Unmarshal([]byte(testCase.Input), &data)
			assert.NilError(t, err, testCase.Name)

			assert.DeepEqual(t, testCase.Expect, data)
		})
	}
}
