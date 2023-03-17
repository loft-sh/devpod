package smoke

import (
	"github.com/loft-sh/devpod/test/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = RegisterSmokeTestCase("local provider", func() {
	ginkgo.It("devpod up", func() {
		f := framework.Framework{}
		err := f.ExecCommand([]string{"devpod", "up"})
		if err != nil {
			ginkgo.Fail(err.Error())
		}
	})
})
