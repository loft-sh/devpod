package git

import (
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

type testCaseNormalizeRepository struct {
	in             string
	expectedRepo   string
	expectedBranch string
}

func TestNormalizeRepository(t *testing.T) {
	testCases := []testCaseNormalizeRepository{
		{
			in:             "ssh://github.com/loft-sh/devpod.git",
			expectedRepo:   "ssh://github.com/loft-sh/devpod.git",
			expectedBranch: "",
		},
		{
			in:             "git@github.com/loft-sh/devpod-without-branch.git",
			expectedRepo:   "git@github.com/loft-sh/devpod-without-branch.git",
			expectedBranch: "",
		},
		{
			in:             "https://github.com/loft-sh/devpod.git",
			expectedRepo:   "https://github.com/loft-sh/devpod.git",
			expectedBranch: "",
		},
		{
			in:             "github.com/loft-sh/devpod.git",
			expectedRepo:   "https://github.com/loft-sh/devpod.git",
			expectedBranch: "",
		},
		{
			in:             "github.com/loft-sh/devpod.git@test-branch",
			expectedRepo:   "https://github.com/loft-sh/devpod.git",
			expectedBranch: "test-branch",
		},
		{
			in:             "git@github.com/loft-sh/devpod-with-branch.git@test-branch",
			expectedRepo:   "git@github.com/loft-sh/devpod-with-branch.git",
			expectedBranch: "test-branch",
		},
		{
			in:             "github.com/loft-sh/devpod-without-protocol-with-slash.git@user/branch",
			expectedRepo:   "https://github.com/loft-sh/devpod-without-protocol-with-slash.git",
			expectedBranch: "user/branch",
		},
		{
			in:             "git@github.com/loft-sh/devpod-with-slash.git@user/branch",
			expectedRepo:   "git@github.com/loft-sh/devpod-with-slash.git",
			expectedBranch: "user/branch",
		},
	}

	for _, testCase := range testCases {
		outRepo, outBranch := NormalizeRepository(testCase.in)
		assert.Check(t, cmp.Equal(testCase.expectedRepo, outRepo))
		assert.Check(t, cmp.Equal(testCase.expectedBranch, outBranch))
	}
}
