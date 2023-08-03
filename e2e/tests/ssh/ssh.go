package ssh

import (
	"context"
	"os"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod ssh test suite", func() {
	ctx := context.Background()
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ginkgo.It("should start a new workspace with a docker provider (default) and ssh into it", func() {
		tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/local-test")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")
		_ = f.DevPodProviderAdd(ctx, "docker")
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

		// Start up devpod workspace
		devpodUpDeadline := time.Now().Add(5 * time.Minute)
		devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
		defer cancel()
		err = f.DevPodUp(devpodUpCtx, tempDir)
		framework.ExpectNoError(err)

		devpodSSHDeadline := time.Now().Add(20 * time.Second)
		devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
		defer cancelSSH()
		err = f.DevPodSSHEchoTestString(devpodSSHCtx, tempDir)
		framework.ExpectNoError(err)
	})
})
