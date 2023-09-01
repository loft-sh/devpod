package encoding

import (
	"testing"

	"gotest.tools/assert"
)

type testCase struct {
	name string
	in   []string
	max  int

	expected string
}

func Test(t *testing.T) {
	testCases := []testCase{
		{
			name: "simple",
			in:   []string{"test", "test2", "test3"},

			expected: "test-test2-test3",
		},
		{
			name: "minimal",
			in:   []string{"tes", "test2", "test3"},
			max:  10,

			expected: "tes-0ce0f6",
		},
	}

	for _, testCase := range testCases {
		out := ""
		if testCase.max > 0 {
			out = SafeConcatNameMax(testCase.in, testCase.max)
		} else {
			out = SafeConcatNameMax(testCase.in, MachineUIDLength)
		}

		assert.Equal(t, testCase.expected, out, "unequal in %s", testCase.name)
	}
}
