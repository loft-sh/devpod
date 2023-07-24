package build

import (
	"context"
	"os"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/docker"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod build test suite", func() {
	var initialDir string
	var dockerHelper *docker.DockerHelper

	ginkgo.BeforeEach(func() {
		var err error
		initialDir, err = os.Getwd()
		framework.ExpectNoError(err)
		dockerHelper = &docker.DockerHelper{DockerCommand: "docker"}
	})

	ginkgo.It("build docker", func() {
		ctx := context.Background()

		f := framework.NewDefaultFramework(initialDir + "/bin")
		tempDir, err := framework.CopyToTempDir("tests/build/testdata/docker")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		_ = f.DevPodProviderDelete(ctx, "docker")
		err = f.DevPodProviderAdd(ctx, "docker")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		// do the build
		err = f.DevPodBuild(ctx, tempDir, "--platform", "linux/amd64,linux/arm64", "--repository", "test-repo", "--skip-push")
		framework.ExpectNoError(err)

		// make sure images are there
		_, err = dockerHelper.InspectImage(ctx, "test-repo:devpod-dc8184ef6bc1e01650d714624e640101", false)
		framework.ExpectNoError(err)
		_, err = dockerHelper.InspectImage(ctx, "test-repo:devpod-db2ba9a28c065a6fa970268fbc2eae11", false)
		framework.ExpectNoError(err)
	})
})
