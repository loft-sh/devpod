package up

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	ginkgo.Context("testing up command", ginkgo.Label("up-docker-build"), ginkgo.Ordered, func() {
		var dockerHelper *docker.DockerHelper
		var initialDir string

		ginkgo.BeforeEach(func() {
			var err error
			initialDir, err = os.Getwd()
			framework.ExpectNoError(err)

			dockerHelper = &docker.DockerHelper{DockerCommand: "docker", Log: log.Default}
			framework.ExpectNoError(err)
		})
		ginkgo.Context("with docker", ginkgo.Ordered, func() {
			ginkgo.It("should start a new workspace with multistage build", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-with-multi-stage-build")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir, "--debug")
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(framework.GetTimeout()*3))
			ginkgo.Context("should start a workspace from a Dockerfile build", func() {
				ginkgo.It("should rebuild image in case of changes in files in build context", func(ctx context.Context) {
					tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-dockerfile-buildcontext")
					framework.ExpectNoError(err)
					ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

					f := framework.NewDefaultFramework(initialDir + "/bin")

					_ = f.DevPodProviderDelete(ctx, "docker")
					err = f.DevPodProviderAdd(ctx, "docker")
					framework.ExpectNoError(err)
					err = f.DevPodProviderUse(context.Background(), "docker")
					framework.ExpectNoError(err)

					ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

					// Wait for devpod workspace to come online (deadline: 30s)
					err = f.DevPodUp(ctx, tempDir)
					framework.ExpectNoError(err)

					workspace, err := f.FindWorkspace(ctx, tempDir)
					framework.ExpectNoError(err)

					container, err := dockerHelper.FindDevContainer(ctx, []string{
						fmt.Sprintf("%s=%s", config.DockerIDLabel, workspace.UID),
					})
					framework.ExpectNoError(err)

					image1 := container.Config.LegacyImage

					scriptFile, err := os.OpenFile(tempDir+"/scripts/alias.sh",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
					framework.ExpectNoError(err)

					defer scriptFile.Close()

					ginkgo.By("Changing a file within the context")
					_, err = scriptFile.Write([]byte("alias yr='date +%Y'"))
					framework.ExpectNoError(err)

					ginkgo.By("Starting DevPod again with --recreate")
					err = f.DevPodUp(ctx, tempDir, "--debug", "--recreate")
					framework.ExpectNoError(err)

					container, err = dockerHelper.FindDevContainer(ctx, []string{
						fmt.Sprintf("%s=%s", config.DockerIDLabel, workspace.UID),
					})
					framework.ExpectNoError(err)

					image2 := container.Config.LegacyImage

					gomega.Expect(image2).ShouldNot(gomega.Equal(image1), "images should be different")
				}, ginkgo.SpecTimeout(framework.GetTimeout()))
				ginkgo.It("should not rebuild image for changes in files mentioned in .dockerignore", func(ctx context.Context) {
					tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-dockerfile-buildcontext")
					framework.ExpectNoError(err)
					ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

					f := framework.NewDefaultFramework(initialDir + "/bin")

					_ = f.DevPodProviderDelete(ctx, "docker")
					err = f.DevPodProviderAdd(ctx, "docker")
					framework.ExpectNoError(err)
					err = f.DevPodProviderUse(context.Background(), "docker")
					framework.ExpectNoError(err)

					ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

					// Wait for devpod workspace to come online (deadline: 30s)
					err = f.DevPodUp(ctx, tempDir)
					framework.ExpectNoError(err)

					workspace, err := f.FindWorkspace(ctx, tempDir)
					framework.ExpectNoError(err)

					container, err := dockerHelper.FindDevContainer(ctx, []string{
						fmt.Sprintf("%s=%s", config.DockerIDLabel, workspace.UID),
					})
					framework.ExpectNoError(err)

					image1 := container.Config.LegacyImage

					scriptFile, err := os.OpenFile(tempDir+"/scripts/install.sh",
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
					framework.ExpectNoError(err)

					defer scriptFile.Close()

					ginkgo.By("Changing a file within context")
					_, err = scriptFile.Write([]byte("apt install python"))
					framework.ExpectNoError(err)

					ginkgo.By("Starting DevPod again with --recreate")
					err = f.DevPodUp(ctx, tempDir, "--debug", "--recreate")
					framework.ExpectNoError(err)

					container, err = dockerHelper.FindDevContainer(ctx, []string{
						fmt.Sprintf("%s=%s", config.DockerIDLabel, workspace.UID),
					})
					framework.ExpectNoError(err)

					image2 := container.Config.LegacyImage

					gomega.Expect(image2).Should(gomega.Equal(image1), "image should be same")
				}, ginkgo.SpecTimeout(framework.GetTimeout()))
			})
		})
	})
})
