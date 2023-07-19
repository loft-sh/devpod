package ide

import (
	"context"
	"os"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod ide test suite", func() {
	var initialDir string

	ginkgo.BeforeEach(func() {
		var err error
		initialDir, err = os.Getwd()
		framework.ExpectNoError(err)
	})

	ginkgo.It("start ides", func() {
		ctx := context.Background()

		f := framework.NewDefaultFramework(initialDir + "/bin")
		tempDir, err := framework.CopyToTempDir("tests/ide/testdata")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		_ = f.DevPodProviderDelete(ctx, "docker")
		err = f.DevPodProviderAdd(ctx, "docker")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

		err = f.DevPodUpWithIDE(ctx, tempDir, "--open-ide=false", "--ide=vscode")
		framework.ExpectNoError(err)

		err = f.DevPodUpWithIDE(ctx, tempDir, "--open-ide=false", "--ide=openvscode")
		framework.ExpectNoError(err)

		err = f.DevPodUpWithIDE(ctx, tempDir, "--open-ide=false", "--ide=jupyternotebook")
		framework.ExpectNoError(err)

		err = f.DevPodUpWithIDE(ctx, tempDir, "--open-ide=false", "--ide=fleet")
		framework.ExpectNoError(err)

		// check if ssh works
		err = f.DevPodSSHEchoTestString(ctx, tempDir)
		framework.ExpectNoError(err)

		// TODO: test jetbrains ides
	})
})
