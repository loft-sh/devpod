package smoke

import (
	"github.com/loft-sh/devpod/test/e2e/framework"
)

func RegisterSmokeTestCase(testcase string, testfunction func()) bool {
	return framework.RegisterTestCase("smoke", testcase, testfunction)
}
