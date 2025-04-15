package up

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/log"
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

			dockerHelper = &docker.DockerHelper{DockerCommand: "docker", Log: log.Default}
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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
			ginkgo.It("should start a new workspace with existing running container", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/no-devcontainer")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				err = dockerHelper.Run(ctx, []string{"run", "-d", "--label", "devpod-e2e-test-container=true", "-w", "/workspaces/e2e", "mcr.microsoft.com/vscode/devcontainers/base:alpine", "sleep", "infinity"}, nil, nil, nil)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					"devpod-e2e-test-container=true",
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 container is created")
				ginkgo.DeferCleanup(dockerHelper.Remove, ids[0])
				ginkgo.DeferCleanup(dockerHelper.Stop, ids[0])

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Config.WorkingDir).To(gomega.Equal("/workspaces/e2e"))

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir, "--source", fmt.Sprintf("container:%s", containerDetail.ID))
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

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
				}, ginkgo.SpecTimeout(framework.GetTimeout()))
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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with dotfiles - no install script, commit", func(ctx context.Context) {
				// need to debug
				if runtime.GOOS == "windows" {
					ginkgo.Skip("skipping on windows")
				}

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
				err = f.DevPodUp(ctx, tempDir, "--dotfiles", "https://github.com/loft-sh/example-dotfiles@sha256:9a0b41808bf8f50e9871b3b5c9280fe22bf46a04")
				framework.ExpectNoError(err)

				out, err := f.DevPodSSH(ctx, tempDir, "ls ~/.file*")
				framework.ExpectNoError(err)

				expectedOutput := `/home/vscode/.file1
/home/vscode/.file2
/home/vscode/.file3
`
				framework.ExpectEqual(out, expectedOutput, "should match")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
			ginkgo.It("should start a new workspace with dotfiles - no install script, branch", func(ctx context.Context) {
				// need to debug
				if runtime.GOOS == "windows" {
					ginkgo.Skip("skipping on windows")
				}

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
				err = f.DevPodUp(ctx, tempDir, "--dotfiles", "https://github.com/loft-sh/example-dotfiles@do-not-delete")
				framework.ExpectNoError(err)

				out, err := f.DevPodSSH(ctx, tempDir, "cat ~/.branch_test")
				framework.ExpectNoError(err)

				expectedOutput := "test\n"
				framework.ExpectEqual(out, expectedOutput, "should match")
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

			ginkgo.It("should start a new workspace with custom image", func(ctx context.Context) {
				if runtime.GOOS == "windows" {
					ginkgo.Skip("skipping on windows")
				}

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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))

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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
		})
	})
})
