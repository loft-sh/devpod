package git

import (
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

type testCaseNormalizeRepository struct {
	in                  string
	expectedPRReference string
	expectedRepo        string
	expectedBranch      string
	expectedCommit      string
	expectedSubpath     string
}

type testCaseGetBranchNameForPR struct {
	in             string
	expectedBranch string
}

func TestNormalizeRepository(t *testing.T) {
	testCases := []testCaseNormalizeRepository{
		{
			in:                  "ssh://github.com/loft-sh/devpod.git",
			expectedRepo:        "ssh://github.com/loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "ssh://git@github.com/loft-sh/devpod.git",
			expectedRepo:        "ssh://git@github.com/loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "git@github.com/loft-sh/devpod-without-branch.git",
			expectedRepo:        "git@github.com/loft-sh/devpod-without-branch.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "https://github.com/loft-sh/devpod.git",
			expectedRepo:        "https://github.com/loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "github.com/loft-sh/devpod.git",
			expectedRepo:        "https://github.com/loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "github.com/loft-sh/devpod.git@test-branch",
			expectedRepo:        "https://github.com/loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "test-branch",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "git@github.com/loft-sh/devpod-with-branch.git@test-branch",
			expectedRepo:        "git@github.com/loft-sh/devpod-with-branch.git",
			expectedPRReference: "",
			expectedBranch:      "test-branch",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "git@github.com/loft-sh/devpod-with-branch.git@test_branch",
			expectedRepo:        "git@github.com/loft-sh/devpod-with-branch.git",
			expectedPRReference: "",
			expectedBranch:      "test_branch",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "ssh://git@github.com/loft-sh/devpod.git@test_branch",
			expectedRepo:        "ssh://git@github.com/loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "test_branch",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "github.com/loft-sh/devpod-without-protocol-with-slash.git@user/branch",
			expectedRepo:        "https://github.com/loft-sh/devpod-without-protocol-with-slash.git",
			expectedPRReference: "",
			expectedBranch:      "user/branch",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "git@github.com/loft-sh/devpod-with-slash.git@user/branch",
			expectedRepo:        "git@github.com/loft-sh/devpod-with-slash.git",
			expectedPRReference: "",
			expectedBranch:      "user/branch",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "github.com/loft-sh/devpod.git@sha256:905ffb0",
			expectedRepo:        "https://github.com/loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "905ffb0",
			expectedSubpath:     "",
		},
		{
			in:                  "git@github.com:loft-sh/devpod.git@sha256:905ffb0",
			expectedRepo:        "git@github.com:loft-sh/devpod.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "905ffb0",
			expectedSubpath:     "",
		},
		{
			in:                  "github.com/loft-sh/devpod.git@pull/996/head",
			expectedRepo:        "https://github.com/loft-sh/devpod.git",
			expectedPRReference: "pull/996/head",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "git@github.com:loft-sh/devpod.git@pull/996/head",
			expectedRepo:        "git@github.com:loft-sh/devpod.git",
			expectedPRReference: "pull/996/head",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "",
		},
		{
			in:                  "github.com/loft-sh/devpod-without-protocol-with-slash.git@subpath:/test/path",
			expectedRepo:        "https://github.com/loft-sh/devpod-without-protocol-with-slash.git",
			expectedPRReference: "",
			expectedBranch:      "",
			expectedCommit:      "",
			expectedSubpath:     "/test/path",
		},
	}

	for _, testCase := range testCases {
		outRepo, outPRReference, outBranch, outCommit, outSubpath := NormalizeRepository(testCase.in)
		assert.Check(t, cmp.Equal(testCase.expectedRepo, outRepo))
		assert.Check(t, cmp.Equal(testCase.expectedPRReference, outPRReference))
		assert.Check(t, cmp.Equal(testCase.expectedBranch, outBranch))
		assert.Check(t, cmp.Equal(testCase.expectedCommit, outCommit))
		assert.Check(t, cmp.Equal(testCase.expectedSubpath, outSubpath))
	}
}

func TestGetBranchNameForPRReference(t *testing.T) {
	testCases := []testCaseGetBranchNameForPR{
		{
			in:             "pull/996/head",
			expectedBranch: "PR996",
		},
		{
			in:             "pull/abc/head",
			expectedBranch: "pull/abc/head",
		},
	}

	for _, testCase := range testCases {
		outBranch := GetBranchNameForPR(testCase.in)
		assert.Check(t, cmp.Equal(testCase.expectedBranch, outBranch))
	}
}
