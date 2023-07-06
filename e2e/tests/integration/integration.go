package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("[integration]: devpod provider ssh test suite", ginkgo.Ordered, func() {
	var initialDir string
	ctx := context.Background()

	ginkgo.BeforeEach(func() {
		var err error
		initialDir, err = os.Getwd()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should generate ssh keypairs", func() {
		_, err := os.Stat(os.Getenv("HOME") + "/.ssh/id_rsa")
		if err != nil {
			fmt.Println("generating ssh keys")
			cmd := exec.Command("ssh-keygen", "-q", "-t", "rsa", "-N", "", "-f", os.Getenv("HOME")+"/.ssh/id_rsa")
			err = cmd.Run()
			framework.ExpectNoError(err)

			cmd = exec.Command("ssh-keygen", "-y", "-f", os.Getenv("HOME")+"/.ssh/id_rsa")
			output, err := cmd.Output()
			framework.ExpectNoError(err)

			err = os.WriteFile(os.Getenv("HOME")+"/.ssh/id_rsa.pub", output, 0600)
			framework.ExpectNoError(err)
		}

		cmd := exec.Command("ssh-keygen", "-y", "-f", os.Getenv("HOME")+"/.ssh/id_rsa")
		publicKey, err := cmd.Output()
		framework.ExpectNoError(err)

		_, err = os.Stat(os.Getenv("HOME") + "/.ssh/authorized_keys")
		if err != nil {
			err = os.WriteFile(os.Getenv("HOME")+"/.ssh/authorized_keys", publicKey, 0600)
			framework.ExpectNoError(err)
		} else {
			f, err := os.OpenFile(os.Getenv("HOME")+"/.ssh/authorized_keys",
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			framework.ExpectNoError(err)

			defer f.Close()
			_, err = f.Write(publicKey)
			framework.ExpectNoError(err)
		}
	})

	ginkgo.It("should add provider to devpod", func() {
		f := framework.NewDefaultFramework(initialDir + "/bin")
		// ensure we don't have the ssh provider present
		err := f.DevPodProviderDelete(ctx, "ssh")
		if err != nil {
			fmt.Println("warning: " + err.Error())
		}

		err = f.DevPodProviderAdd(ctx, "ssh", "-o", "HOST=localhost")
		framework.ExpectNoError(err)
	})

	ginkgo.It("should run devpod up", func(ctx context.Context) {
		f := framework.NewDefaultFramework(initialDir + "/bin")
		err := f.DevPodUp(ctx, "tests/integration/testdata/")
		framework.ExpectNoError(err)
	})

	ginkgo.It("should run commands to workspace via ssh", func() {
		cmd := exec.Command("ssh", "testdata.devpod", "echo", "test")
		output, err := cmd.Output()
		framework.ExpectNoError(err)

		gomega.Expect(output).To(gomega.Equal([]byte("test\n")))
	})

	ginkgo.It("should cleanup devpod workspace", func(ctx context.Context) {
		f := framework.NewDefaultFramework(initialDir + "/bin")
		err := f.DevPodWorkspaceDelete(ctx, "testdata")
		framework.ExpectNoError(err)
	})
})
