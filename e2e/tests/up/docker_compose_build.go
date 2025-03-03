package up

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/compose"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	ginkgo.Context("testing up command", ginkgo.Label("up-docker-compose-build"), ginkgo.Ordered, func() {
		var dockerHelper *docker.DockerHelper
		var composeHelper *compose.ComposeHelper
		var initialDir string

		ginkgo.BeforeEach(func() {
			var err error
			initialDir, err = os.Getwd()
			framework.ExpectNoError(err)

			dockerHelper = &docker.DockerHelper{DockerCommand: "docker", Log: log.Default}
			composeHelper, err = compose.NewComposeHelper("", dockerHelper)
			framework.ExpectNoError(err)
		})

		ginkgo.Context("with docker-compose", func() {
			ginkgo.It("should start a new workspace with multistage build", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-with-multi-stage-build")
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

			ginkgo.Context("with --recreate", func() {
				ginkgo.It("should NOT delete container when rebuild fails", func(ctx context.Context) {
					tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-rebuild-fail")
					framework.ExpectNoError(err)
					ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

					f := framework.NewDefaultFramework(initialDir + "/bin")
					_ = f.DevPodProviderAdd(ctx, "docker")
					err = f.DevPodProviderUse(ctx, "docker")
					framework.ExpectNoError(err)

					ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

					ginkgo.By("Starting DevPod")
					err = f.DevPodUp(ctx, tempDir)
					framework.ExpectNoError(err)

					workspace, err := f.FindWorkspace(ctx, tempDir)
					framework.ExpectNoError(err)

					ginkgo.By("Should start a docker-compose container")
					ids, err := dockerHelper.FindContainer(ctx, []string{
						fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
						fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
					})
					framework.ExpectNoError(err)
					gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

					ginkgo.By("Modifying .devcontainer.json with failing changes")
					origPath := filepath.Join(tempDir, ".devcontainer.json")
					err = os.Remove(origPath)
					framework.ExpectNoError(err)

					failingConfig, err := os.Open(filepath.Join(tempDir, "fail.devcontainer.json"))
					framework.ExpectNoError(err)

					newConfig, err := os.Create(origPath)
					framework.ExpectNoError(err)

					_, err = io.Copy(newConfig, failingConfig)
					framework.ExpectNoError(err)

					ginkgo.By("Starting DevPod again with --recreate")
					err = f.DevPodUp(ctx, tempDir, "--debug", "--recreate")
					framework.ExpectError(err)

					ginkgo.By("Should leave original container running")
					ids2, err := dockerHelper.FindContainer(ctx, []string{
						fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
						fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
					})
					framework.ExpectNoError(err)
					gomega.Expect(ids2[0]).To(gomega.Equal(ids[0]), "Should use original container")
				})

				ginkgo.It("should delete container upon successful rebuild", func(ctx context.Context) {
					tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-rebuild-success")
					framework.ExpectNoError(err)
					ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

					f := framework.NewDefaultFramework(initialDir + "/bin")
					_ = f.DevPodProviderAdd(ctx, "docker")
					err = f.DevPodProviderUse(ctx, "docker")
					framework.ExpectNoError(err)

					ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

					ginkgo.By("Starting DevPod")
					err = f.DevPodUp(ctx, tempDir)
					framework.ExpectNoError(err)

					workspace, err := f.FindWorkspace(ctx, tempDir)
					framework.ExpectNoError(err)

					ginkgo.By("Should start a docker-compose container")
					ids, err := dockerHelper.FindContainer(ctx, []string{
						fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
						fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
					})
					framework.ExpectNoError(err)
					gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

					ginkgo.By("Starting DevPod again with --recreate")
					err = f.DevPodUp(ctx, tempDir, "--debug", "--recreate")
					framework.ExpectNoError(err)

					ginkgo.By("Should start a new docker-compose container on rebuild")
					ids2, err := dockerHelper.FindContainer(ctx, []string{
						fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
						fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
					})
					framework.ExpectNoError(err)
					gomega.Expect(ids2[0]).NotTo(gomega.Equal(ids[0]), "Should restart container")
				})

			})
		})
	})
})
