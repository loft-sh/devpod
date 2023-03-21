package ssh

import (
	"context"
	"os"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod ssh test suite", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ginkgo.It("should start a new workspace with a local provider (default) and ssh into it", func() {
		tempDir, err := framework.CopyToTempDir("tests/up/testdata/local-test")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")

		f.DevPodProviderUse(context.Background(), "local")

		// Start up devpod workspace
		devpodUpDeadline := time.Now().Add(2 * time.Minute)
		devpodUpCtx, devpodUpCancel := context.WithDeadline(context.Background(), devpodUpDeadline)
		go func(ctx context.Context) {
			// Wait for devpod workspace to come online (dealine: 1 minute)
			f.DevPodUp(devpodUpCtx, tempDir)
		}(devpodUpCtx)

		// Wait 30 seconds for workspace to come online
		time.Sleep(20 * time.Second)

		devpodSSHDeadline := time.Now().Add(20 * time.Second)
		devpodSSHCtx, _ := context.WithDeadline(context.Background(), devpodSSHDeadline)

		defer devpodUpCancel()
		err = f.DevPodSSHEchoTestString(devpodSSHCtx, tempDir)
		framework.ExpectNoError(err)

	})
})
