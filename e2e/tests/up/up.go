package up

import (
	"context"
	"os"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ginkgo.It("should start a new workspace with a local provider (default)", func() {
		tempDir, err := framework.CopyToTempDir("tests/up/testdata/local-test")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")
		_ = f.DevPodProviderAdd([]string{"docker"})
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		// Wait for devpod workspace to come online (dealine: 30s)
		deadline := time.Now().Add(30 * time.Second)
		devpodUpCtx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()
		err = f.DevPodUp(devpodUpCtx, tempDir)
		framework.ExpectNoError(err)
	})
})
