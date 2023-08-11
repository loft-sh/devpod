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
	expectedCommit string
}

func TestNormalizeRepository(t *testing.T) {
	testCases := []testCaseNormalizeRepository{
		{
			in:             "ssh://github.com/loft-sh/devpod.git",
			expectedRepo:   "ssh://github.com/loft-sh/devpod.git",
			expectedBranch: "",
			expectedCommit: "",
		},
		{
			in:             "git@github.com/loft-sh/devpod-without-branch.git",
			expectedRepo:   "git@github.com/loft-sh/devpod-without-branch.git",
			expectedBranch: "",
			expectedCommit: "",
		},
		{
			in:             "https://github.com/loft-sh/devpod.git",
			expectedRepo:   "https://github.com/loft-sh/devpod.git",
			expectedBranch: "",
			expectedCommit: "",
		},
		{
			in:             "github.com/loft-sh/devpod.git",
			expectedRepo:   "https://github.com/loft-sh/devpod.git",
			expectedBranch: "",
			expectedCommit: "",
		},
		{
			in:             "github.com/loft-sh/devpod.git@test-branch",
			expectedRepo:   "https://github.com/loft-sh/devpod.git",
			expectedBranch: "test-branch",
			expectedCommit: "",
		},
		{
			in:             "git@github.com/loft-sh/devpod-with-branch.git@test-branch",
			expectedRepo:   "git@github.com/loft-sh/devpod-with-branch.git",
			expectedBranch: "test-branch",
			expectedCommit: "",
		},
		{
			in:             "git@github.com/loft-sh/devpod-with-branch.git@test_branch",
			expectedRepo:   "git@github.com/loft-sh/devpod-with-branch.git",
			expectedBranch: "test_branch",
			expectedCommit: "",
		},
		{
			in:             "github.com/loft-sh/devpod-without-protocol-with-slash.git@user/branch",
			expectedRepo:   "https://github.com/loft-sh/devpod-without-protocol-with-slash.git",
			expectedBranch: "user/branch",
			expectedCommit: "",
		},
		{
			in:             "git@github.com/loft-sh/devpod-with-slash.git@user/branch",
			expectedRepo:   "git@github.com/loft-sh/devpod-with-slash.git",
			expectedBranch: "user/branch",
			expectedCommit: "",
		},
		{
			in:             "github.com/loft-sh/devpod.git@sha256:905ffb0",
			expectedRepo:   "https://github.com/loft-sh/devpod.git",
			expectedBranch: "",
			expectedCommit: "905ffb0",
		},
		{
			in:             "git@github.com:loft-sh/devpod.git@sha256:905ffb0",
			expectedRepo:   "git@github.com:loft-sh/devpod.git",
			expectedBranch: "",
			expectedCommit: "905ffb0",
		},
	}

	for _, testCase := range testCases {
		outRepo, outBranch, outCommit := NormalizeRepository(testCase.in)
		assert.Check(t, cmp.Equal(testCase.expectedRepo, outRepo))
		assert.Check(t, cmp.Equal(testCase.expectedBranch, outBranch))
		assert.Check(t, cmp.Equal(testCase.expectedCommit, outCommit))
	}
}
