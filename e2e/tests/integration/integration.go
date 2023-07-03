package integration

import (
	"fmt"
	"os/exec"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("[integration]: devpod provider ssh test suite", ginkgo.Ordered, func() {

	ginkgo.FIt("should add provider to devpod", func() {
		// ensure we don't have the ssh provider present
		cmd := exec.Command("bin/devpod-linux-amd64", "provider", "delete", "ssh")
		err := cmd.Run()
		if err != nil {
			fmt.Println("warning: " + err.Error())
		}

		cmd = exec.Command("bin/devpod-linux-amd64", "provider", "add", "ssh", "-o", "HOST=localhost")
		err = cmd.Run()
		framework.ExpectNoError(err)
	})

	ginkgo.FIt("should run devpod up", func() {
		// ensure we don't have the ssh provider present
		cmd := exec.Command("bin/devpod-linux-amd64", "up", "--debug", "--ide=none", "tests/integration/testdata/")
		err := cmd.Run()
		framework.ExpectNoError(err)
	})

	ginkgo.FIt("should run commands to workspace via ssh", func() {
		// ensure we don't have the ssh provider present
		cmd := exec.Command("ssh", "testdata.devpod", "echo", "test")
		output, err := cmd.Output()
		framework.ExpectNoError(err)

		gomega.Expect(output).To(gomega.Equal([]byte("test\n")))
	})

	ginkgo.FIt("should cleanup devpod workspace", func() {
		// ensure we don't have the ssh provider present
		cmd := exec.Command("bin/devpod-linux-amd64", "delete", "--debug", "--force", "testdata")
		err := cmd.Run()
		framework.ExpectNoError(err)
	})
})
