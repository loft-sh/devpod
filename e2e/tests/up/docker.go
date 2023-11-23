package up

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	ginkgo.Context("testing up command", ginkgo.Label("up-docker"), ginkgo.Ordered, func() {
		var dockerHelper *docker.DockerHelper
		var initialDir string

		ginkgo.BeforeEach(func() {
			var err error
			initialDir, err = os.Getwd()
			framework.ExpectNoError(err)

			dockerHelper = &docker.DockerHelper{DockerCommand: "docker"}
			framework.ExpectNoError(err)
		})

		ginkgo.Context("with docker", ginkgo.Ordered, func() {
			ginkgo.It("should start a new workspace with existing image", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(framework.GetTiemout()))
			ginkgo.It("should start a new workspace and substitute devcontainer.json variables", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-variables")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				projectName := workspace.ID
				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", config.DockerIDLabel, workspace.UID),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				devContainerID, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/dev-container-id.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(devContainerID).NotTo(gomega.BeEmpty())

				containerEnvPath, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-env-path.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerEnvPath).To(gomega.ContainSubstring("/usr/local/bin"))

				localEnvHome, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-env-home.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(localEnvHome).To(gomega.Equal(os.Getenv("HOME")))

				localWorkspaceFolder, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-workspace-folder.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(framework.CleanString(localWorkspaceFolder)).To(gomega.Equal(framework.CleanString(tempDir)))

				localWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(localWorkspaceFolderBasename).To(gomega.Equal(filepath.Base(tempDir)))

				containerWorkspaceFolder, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(framework.CleanString(containerWorkspaceFolder)).To(gomega.Equal(
					framework.CleanString("workspaces" + filepath.Base(tempDir)),
				))

				containerWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerWorkspaceFolderBasename).To(gomega.Equal(filepath.Base(tempDir)))
			}, ginkgo.SpecTimeout(framework.GetTiemout()))

			ginkgo.It("should start a new workspace with mounts", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-mounts")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir, "--debug")
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", config.DockerIDLabel, workspace.UID),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				foo, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/mnt1/foo.txt", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(foo).To(gomega.Equal("BAR"))

				bar, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/mnt2/bar.txt", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(bar).To(gomega.Equal("FOO"))
			}, ginkgo.SpecTimeout(framework.GetTiemout()))

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
			}, ginkgo.SpecTimeout(framework.GetTiemout()*3))

			ginkgo.Context("should start a new workspace with features", func() {
				ginkgo.It("ensure dependencies installed via features are accessible in lifecycle hooks", func(ctx context.Context) {
					tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-features-lifecycle-hooks")
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
				}, ginkgo.SpecTimeout(framework.GetTiemout()))
			})
			ginkgo.It("should start a new workspace with dotfiles - no install script", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir, "--dotfiles", "https://github.com/loft-sh/example-dotfiles")
				framework.ExpectNoError(err)

				out, err := f.DevPodSSH(ctx, tempDir, "ls ~/.file*")
				framework.ExpectNoError(err)

				expectedOutput := `/home/vscode/.file1
/home/vscode/.file2
/home/vscode/.file3
`
				framework.ExpectEqual(out, expectedOutput, "should match")
			}, ginkgo.SpecTimeout(framework.GetTiemout()))
			ginkgo.It("should start a new workspace with dotfiles - install script", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir, "--dotfiles", "https://github.com/loft-sh/example-dotfiles", "--dotfiles-script", "install-example")
				framework.ExpectNoError(err)

				out, err := f.DevPodSSH(ctx, tempDir, "ls /tmp/worked")
				framework.ExpectNoError(err)

				expectedOutput := "/tmp/worked\n"

				framework.ExpectEqual(out, expectedOutput, "should match")
			}, ginkgo.SpecTimeout(framework.GetTiemout()))

			ginkgo.It("should start a new workspace with custom image", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir, "--devcontainer-image", "mcr.microsoft.com/vscode/devcontainers/base:alpine")
				framework.ExpectNoError(err)

				out, err := f.DevPodSSH(ctx, tempDir, "grep ^ID= /etc/os-release")
				framework.ExpectNoError(err)

				expectedOutput := "ID=alpine\n"
				unexpectedOutput := "ID=debian\n"

				framework.ExpectEqual(out, expectedOutput, "should match")
				framework.ExpectNotEqual(out, unexpectedOutput, "should NOT match")
			}, ginkgo.SpecTimeout(framework.GetTiemout()))
			ginkgo.It("should start a new workspace with custom image and skip building", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-with-multi-stage-build")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir, "--devcontainer-image", "mcr.microsoft.com/vscode/devcontainers/base:alpine")
				framework.ExpectNoError(err)

				out, err := f.DevPodSSH(ctx, tempDir, "grep ^ID= /etc/os-release")
				framework.ExpectNoError(err)

				expectedOutput := "ID=alpine\n"
				unexpectedOutput := "ID=debian\n"

				framework.ExpectEqual(out, expectedOutput, "should match")
				framework.ExpectNotEqual(out, unexpectedOutput, "should NOT match")
			}, ginkgo.SpecTimeout(framework.GetTiemout()))

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
				}, ginkgo.SpecTimeout(framework.GetTiemout()))
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
				}, ginkgo.SpecTimeout(framework.GetTiemout()))
			})
			ginkgo.It("should use http headers to download feature", func(ctx context.Context) {
				server := ghttp.NewServer()

				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-features-http-headers")
				framework.ExpectNoError(err)

				featureArchiveFilePath := path.Join(tempDir, "devcontainer-feature-hello.tgz")
				featureFiles := []string{path.Join(tempDir, "devcontainer-feature.json"), path.Join(tempDir, "install.sh")}
				err = createTarGzArchive(featureArchiveFilePath, featureFiles)
				framework.ExpectNoError(err)

				devContainerFileBuf, err := os.ReadFile(path.Join(tempDir, ".devcontainer.json"))
				framework.ExpectNoError(err)

				output := strings.Replace(string(devContainerFileBuf), "#{server_url}", server.URL(), -1)
				err = os.WriteFile(path.Join(tempDir, ".devcontainer.json"), []byte(output), 0644)
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)
				ginkgo.DeferCleanup(server.Close)

				respHeader := http.Header{}
				respHeader.Set("Content-Disposition", "attachment; filename=devcontainer-feature-hello.tgz")

				featureArchiveFileBuf, err := os.ReadFile(featureArchiveFilePath)
				framework.ExpectNoError(err)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/devcontainer-feature-hello.tgz"),
						ghttp.VerifyHeaderKV("Foo-Header", "Foo"),
						ghttp.RespondWith(http.StatusOK, featureArchiveFileBuf, respHeader),
					),
				)

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)
				server.Close()
			}, ginkgo.SpecTimeout(framework.GetTiemout()))
		})
	})
})
