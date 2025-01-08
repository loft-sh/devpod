package util

import (
	"os"
	"runtime"
	"testing"

	"gotest.tools/assert"
)

func TestUserHomeDir(t *testing.T) {
	// Remember to reset environment variables after the test
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	})

	type input struct {
		home, userProfile string
	}

	type testCase struct {
		Name   string
		Input  input
		Expect string
	}

	testCases := []testCase{
		{
			// $HOME is preferred on every platform
			Name: "both HOME and USERPROFILE are set",
			Input: input{
				home:        "home",
				userProfile: "userProfile",
			},
			Expect: "home",
		},
	}
	if runtime.GOOS == "windows" {
		// On Windows, after $HOME, %userprofile% value is checked
		testCases = append(testCases, testCase{
			Name: "HOME is unset and USERPROFILE is set",
			Input: input{
				home:        "",
				userProfile: "userProfile",
			},
			Expect: "userProfile",
		})
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			os.Setenv("HOME", test.Input.home)
			os.Setenv("USERPROFILE", test.Input.userProfile)

			got, err := UserHomeDir()
			assert.NilError(t, err, test.Name)
			assert.Equal(t, test.Expect, got)
		})
	}
}
