package up

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/loft-sh/devpod/pkg/compose"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	var dockerHelper *docker.DockerHelper
	var composeHelper *compose.ComposeHelper
	var initialDir string

	ginkgo.BeforeEach(func() {
		var err error
		initialDir, err = os.Getwd()
		framework.ExpectNoError(err)

		dockerHelper = &docker.DockerHelper{DockerCommand: "docker"}
		composeHelper, err = compose.NewComposeHelper("", dockerHelper)
		framework.ExpectNoError(err)
	})

	ginkgo.It("with env vars", func() {
		ctx := context.Background()
		f := framework.NewDefaultFramework(initialDir + "/bin")

		_ = f.DevPodProviderDelete(ctx, "docker")
		err := f.DevPodProviderAdd(ctx, "docker")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(ctx, "docker")
		framework.ExpectNoError(err)

		name := "vscode-remote-try-python"
		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), name)

		// Wait for devpod workspace to come online (deadline: 30s)
		err = f.DevPodUp(ctx, "github.com/microsoft/vscode-remote-try-python")
		framework.ExpectNoError(err)

		// check env var
		out, err := f.DevPodSSH(ctx, name, "echo -n $TEST_VAR")
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "", "should be empty")

		// set env var
		value := "test-variable"
		err = f.DevPodUp(ctx, name, "--workspace-env", "TEST_VAR="+value)
		framework.ExpectNoError(err)

		// check env var
		out, err = f.DevPodSSH(ctx, name, "echo -n $TEST_VAR")
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, value, "should be set now")

		// check env var again
		err = f.DevPodUp(ctx, name)
		framework.ExpectNoError(err)

		// check env var
		out, err = f.DevPodSSH(ctx, name, "echo -n $TEST_VAR")
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, value, "should still be set")

		// delete env var
		err = f.DevPodUp(ctx, name, "--workspace-env", "TEST_VAR=")
		framework.ExpectNoError(err)

		// check env var
		out, err = f.DevPodSSH(ctx, name, "echo -n $TEST_VAR")
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "", "should be empty")
	})

	ginkgo.It("should allow checkout of a GitRepo from a commit hash", func() {
		ctx := context.Background()
		f := framework.NewDefaultFramework(initialDir + "/bin")

		_ = f.DevPodProviderDelete(ctx, "docker")
		err := f.DevPodProviderAdd(ctx, "docker")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(ctx, "docker")
		framework.ExpectNoError(err)

		name := "vscode-remote-try-python-sha256-0c1547c"
		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), name)

		// Wait for devpod workspace to come online (deadline: 30s)
		err = f.DevPodUp(ctx, "github.com/microsoft/vscode-remote-try-python@sha256:0c1547c")
		framework.ExpectNoError(err)
	})

	ginkgo.It("run devpod in Kubernetes", func() {
		ctx := context.Background()
		f := framework.NewDefaultFramework(initialDir + "/bin")
		tempDir, err := framework.CopyToTempDir("tests/up/testdata/kubernetes")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		_ = f.DevPodProviderDelete(ctx, "kubernetes")
		err = f.DevPodProviderAdd(ctx, "kubernetes", "-o", "KUBERNETES_NAMESPACE=devpod")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			err = f.DevPodProviderDelete(ctx, "kubernetes")
			framework.ExpectNoError(err)
		})

		// run up
		err = f.DevPodUp(ctx, tempDir)
		framework.ExpectNoError(err)

		// check pod is there
		cmd := exec.Command("kubectl", "get", "pods", "-l", "devpod.sh/created=true", "-o", "json", "-n", "devpod")
		stdout, err := cmd.Output()
		framework.ExpectNoError(err)

		// check if pod is there
		list := &corev1.PodList{}
		err = json.Unmarshal(stdout, list)
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1, "Expect 1 pod")
		framework.ExpectEqual(len(list.Items[0].Spec.Containers), 1, "Expect 1 container")
		framework.ExpectEqual(list.Items[0].Spec.Containers[0].Image, "mcr.microsoft.com/devcontainers/go:0-1.19-bullseye", "Expect container image")

		// check if ssh works
		err = f.DevPodSSHEchoTestString(ctx, tempDir)
		framework.ExpectNoError(err)

		// stop workspace
		err = f.DevPodWorkspaceStop(ctx, tempDir)
		framework.ExpectNoError(err)

		// check pod is there
		cmd = exec.Command("kubectl", "get", "pods", "-l", "devpod.sh/created=true", "-o", "json", "-n", "devpod")
		stdout, err = cmd.Output()
		framework.ExpectNoError(err)

		// check if pod is there
		list = &corev1.PodList{}
		err = json.Unmarshal(stdout, list)
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 0, "Expect no pods")

		// run up
		err = f.DevPodUp(ctx, tempDir)
		framework.ExpectNoError(err)

		// check pod is there
		cmd = exec.Command("kubectl", "get", "pods", "-l", "devpod.sh/created=true", "-o", "json", "-n", "devpod")
		stdout, err = cmd.Output()
		framework.ExpectNoError(err)

		// check if pod is there
		list = &corev1.PodList{}
		err = json.Unmarshal(stdout, list)
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(list.Items), 1, "Expect 1 pod")

		// check if ssh works
		err = f.DevPodSSHEchoTestString(ctx, tempDir)
		framework.ExpectNoError(err)

		// delete workspace
		err = f.DevPodWorkspaceDelete(ctx, tempDir)
		framework.ExpectNoError(err)
	})

	ginkgo.Context("print error message correctly", func() {
		ginkgo.It("make sure devpod output is correct and log-output works correctly", func(ctx context.Context) {
			f := framework.NewDefaultFramework(initialDir + "/bin")
			tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			err = f.DevPodProviderAdd(ctx, "docker", "--name", "test-docker")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				err = f.DevPodProviderDelete(context.Background(), "test-docker")
				framework.ExpectNoError(err)
			})

			err = f.DevPodProviderUse(ctx, "test-docker", "-o", "DOCKER_PATH=abc", "--skip-init")
			framework.ExpectNoError(err)

			// Wait for devpod workspace to come online
			stdout, stderr, err := f.DevPodUpStreams(ctx, tempDir, "--log-output=json")
			deleteErr := f.DevPodWorkspaceDelete(ctx, tempDir, "--force")
			framework.ExpectNoError(deleteErr)
			framework.ExpectError(err, "expected error")
			framework.ExpectNoError(verifyLogStream(strings.NewReader(stdout)))
			framework.ExpectNoError(verifyLogStream(strings.NewReader(stderr)))
			framework.ExpectNoError(findMessage(strings.NewReader(stdout), "exec: \"abc\": executable file not found in $PATH"))
		}, ginkgo.SpecTimeout(60*time.Second))
	})

	ginkgo.Context("cleanup up on failure", func() {
		ginkgo.It("ensure workspace cleanup when failing to create a workspace", func(ctx context.Context) {
			f := framework.NewDefaultFramework(initialDir + "/bin")
			_ = f.DevPodProviderAdd(ctx, "docker")
			err := f.DevPodProviderUse(ctx, "docker")
			framework.ExpectNoError(err)

			initialList, err := f.DevPodList(ctx)
			framework.ExpectNoError(err)
			// Wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, "github.com/i/do-not-exist.git")
			framework.ExpectError(err)

			out, err := f.DevPodList(ctx)
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, initialList)
		}, ginkgo.SpecTimeout(60*time.Second))
		ginkgo.It("ensure workspace cleanup when not a git or folder", func(ctx context.Context) {
			f := framework.NewDefaultFramework(initialDir + "/bin")
			_ = f.DevPodProviderAdd(ctx, "docker")
			err := f.DevPodProviderUse(ctx, "docker")
			framework.ExpectNoError(err)

			initialList, err := f.DevPodList(ctx)
			framework.ExpectNoError(err)
			// Wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, "test1234.com")
			framework.ExpectError(err)

			out, err := f.DevPodList(ctx)
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, initialList)
		}, ginkgo.SpecTimeout(60*time.Second))
	})

	ginkgo.Context("using docker provider", func() {
		ginkgo.Context("with rootfull podman", ginkgo.Ordered, func() {
			ginkgo.It("should setup rootful podman", func(ctx context.Context) {
				wrapper, err := os.Create(initialDir + "/bin/podman-rootful")
				framework.ExpectNoError(err)

				defer wrapper.Close()

				_, err = wrapper.WriteString(`#!/bin/sh
				sudo podman "$@"
				`)
				framework.ExpectNoError(err)

				err = wrapper.Close()
				framework.ExpectNoError(err)

				cmd := exec.Command("sudo", "chmod", "+x", initialDir+"/bin/podman-rootful")
				err = cmd.Run()
				framework.ExpectNoError(err)

				err = exec.Command(initialDir+"/bin/podman-rootful", "ps").Run()
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with existing image", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker", "-o", "DOCKER_PATH="+initialDir+"/bin/podman-rootful")
				framework.ExpectNoError(err)

				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))
		})
		ginkgo.Context("with rootless podman", ginkgo.Ordered, func() {
			ginkgo.It("should start a new workspace with existing image", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")

				_ = f.DevPodProviderDelete(ctx, "docker")
				err = f.DevPodProviderAdd(ctx, "docker", "-o", "DOCKER_PATH=podman")
				framework.ExpectNoError(err)
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				// Wait for devpod workspace to come online (deadline: 30s)
				err = f.DevPodUp(ctx, tempDir)
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))
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
			}, ginkgo.SpecTimeout(60*time.Second))
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
				gomega.Expect(localWorkspaceFolder).To(gomega.Equal(tempDir))

				localWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(localWorkspaceFolderBasename).To(gomega.Equal(filepath.Base(tempDir)))

				containerWorkspaceFolder, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerWorkspaceFolder).To(gomega.Equal(filepath.Join("/workspaces", filepath.Base(tempDir))))

				containerWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerWorkspaceFolderBasename).To(gomega.Equal(filepath.Base(tempDir)))
			}, ginkgo.SpecTimeout(60*time.Second))

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
			}, ginkgo.SpecTimeout(60*time.Second))

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
			}, ginkgo.SpecTimeout(180*time.Second))

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
				}, ginkgo.SpecTimeout(60*time.Second))
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
			}, ginkgo.SpecTimeout(60*time.Second))
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
			}, ginkgo.SpecTimeout(60*time.Second))

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
			}, ginkgo.SpecTimeout(60*time.Second))
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
			}, ginkgo.SpecTimeout(60*time.Second))

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

					image1 := container.Config.Image

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

					image2 := container.Config.Image

					gomega.Expect(image2).ShouldNot(gomega.Equal(image1), "images should be different")
				}, ginkgo.SpecTimeout(60*time.Second))
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

					image1 := container.Config.Image

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

					image2 := container.Config.Image

					gomega.Expect(image2).Should(gomega.Equal(image1), "image should be same")
				}, ginkgo.SpecTimeout(60*time.Second))
			})

		})

		ginkgo.Context("with docker-compose", func() {
			ginkgo.It("should start a new workspace with root folder configuration", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose")
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

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Mounts).To(gomega.HaveLen(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.BeTrue())
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with sub-folder configuration", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-subfolder")
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

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Mounts).To(gomega.HaveLen(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.BeTrue())
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with multiple services", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-multiple-services")
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
				projectName := composeHelper.GetProjectName(workspace.UID)

				appIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(appIDs).To(gomega.HaveLen(1), "app container to be created")

				dbIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "db"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(dbIDs).To(gomega.HaveLen(1), "db container to be created")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with specific services", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-run-services")
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
				projectName := composeHelper.GetProjectName(workspace.UID)

				appIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(appIDs).To(gomega.HaveLen(1), "app container to be created")

				dbIDs, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "db"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(dbIDs).To(gomega.BeEmpty(), "db container not to be created")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with .devcontainer docker-compose overrides", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-overrides")
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
				projectName := composeHelper.GetProjectName(workspace.UID)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, projectName),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Mounts).To(gomega.HaveLen(1), "1 container volume mount")

				mount := containerDetail.Mounts[0]
				gomega.Expect(mount.Source).To(gomega.Equal(tempDir))
				gomega.Expect(mount.Destination).To(gomega.Equal("/workspaces"))
				gomega.Expect(mount.RW).To(gomega.BeTrue())
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with container environment variables set", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-container-env")
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
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "BAR", []string{"ssh", "--command", "echo $FOO", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with container user", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-container-user")
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
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "root", []string{"ssh", "--command", "ps u -p 1", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with privileged", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-privileged")
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

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.Privileged).To(gomega.BeTrue(), "container run with privileged true")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with capAdd", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-capadd")
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

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.CapAdd).To(gomega.ContainElement("SYS_PTRACE"), "image capabilities are not duplicated")
				gomega.Expect(containerDetail.HostConfig.CapAdd).To(gomega.ContainElement("NET_ADMIN"), "devcontainer configuration can add capabilities")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with securityOpt", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-securityOpt")
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

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.HostConfig.SecurityOpt).To(gomega.ContainElement("seccomp=unconfined"), "securityOpts contain seccomp=unconfined")
				gomega.Expect(containerDetail.HostConfig.SecurityOpt).To(gomega.ContainElement("apparmor=unconfined"), "securityOpts contain apparmor=unconfined")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with override command", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-override-command")
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

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				var containerDetails []types.ContainerJSON
				err = dockerHelper.Inspect(ctx, ids, "container", &containerDetails)
				framework.ExpectNoError(err)

				containerDetail := containerDetails[0]
				gomega.Expect(containerDetail.Config.Entrypoint).NotTo(gomega.ContainElement("bash"), "overrides container entry point")
				gomega.Expect(containerDetail.Config.Cmd).To(gomega.BeEmpty(), "overrides container command")
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with remote env", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-remote-env")
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
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "/home/vscode/remote-env.out", []string{"ssh", "--command", "ls $HOME/remote-env.out", projectName})
				framework.ExpectNoError(err)

				err = f.ExecCommand(ctx, true, true, "BAR", []string{"ssh", "--command", "cat $HOME/remote-env.out", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with remote user", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-remote-user")
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
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				err = f.ExecCommand(ctx, true, true, "root", []string{"ssh", "--command", "cat $HOME/remote-user.out", projectName})
				framework.ExpectNoError(err)
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace and substitute devcontainer.json variables", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-variables")
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
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
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
				gomega.Expect(localWorkspaceFolder).To(gomega.Equal(tempDir))

				localWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/local-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(localWorkspaceFolderBasename).To(gomega.Equal(filepath.Base(tempDir)))

				containerWorkspaceFolder, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerWorkspaceFolder).To(gomega.Equal("/workspaces"))

				containerWorkspaceFolderBasename, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/container-workspace-folder-basename.out", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(containerWorkspaceFolderBasename).To(gomega.Equal("workspaces"))
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with mounts", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-mounts")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir, "--debug")
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				_, _, err = f.ExecCommandCapture(ctx, []string{"ssh", "--command", "touch /home/vscode/mnt1/foo.txt", projectName, "--user", "root"})
				framework.ExpectNoError(err)

				_, _, err = f.ExecCommandCapture(ctx, []string{"ssh", "--command", "echo -n BAR > /home/vscode/mnt1/foo.txt", projectName, "--user", "root"})
				framework.ExpectNoError(err)

				foo, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/mnt1/foo.txt", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(foo).To(gomega.Equal("BAR"))

				bar, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/mnt2/bar.txt", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(bar).To(gomega.Equal("FOO"))
			}, ginkgo.SpecTimeout(60*time.Second))

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
			}, ginkgo.SpecTimeout(180*time.Second))

			ginkgo.It("should start a new workspace with host:port forwardPorts", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-forward-ports")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				// Check for docker-compose container running
				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				err = f.DevPodUp(ctx, tempDir, "--debug")
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)
				projectName := workspace.ID

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				done := make(chan error)

				sshContext, sshCancel := context.WithCancel(context.Background())
				go func() {
					cmd := exec.CommandContext(sshContext, "ssh", projectName+".devpod", "sleep", "10")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr

					if err := cmd.Start(); err != nil {
						done <- err
						return
					}

					if err := cmd.Wait(); err != nil {
						done <- err
						return
					}

					done <- nil
				}()

				gomega.Eventually(func(g gomega.Gomega) {
					response, err := http.Get("http://localhost:8080")
					g.Expect(err).NotTo(gomega.HaveOccurred())

					body, err := io.ReadAll(response.Body)
					g.Expect(err).NotTo(gomega.HaveOccurred())
					g.Expect(body).To(gomega.ContainSubstring("Thank you for using nginx."))
				}).
					WithPolling(1 * time.Second).
					WithTimeout(20 * time.Second).
					Should(gomega.Succeed())

				sshCancel()
				err = <-done

				gomega.Expect(err).To(gomega.Or(
					gomega.MatchError("signal: killed"),
					gomega.MatchError(context.Canceled),
				))
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with features", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-features")
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
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

				vclusterVersionOutput, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "vcluster --version", projectName})
				framework.ExpectNoError(err)
				gomega.Expect(vclusterVersionOutput).To(gomega.ContainSubstring("vcluster version 0.15.2"))
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.It("should start a new workspace with env-file", func(ctx context.Context) {
				tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-env-file")
				framework.ExpectNoError(err)
				ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

				f := framework.NewDefaultFramework(initialDir + "/bin")
				_ = f.DevPodProviderAdd(ctx, "docker")
				err = f.DevPodProviderUse(ctx, "docker")
				framework.ExpectNoError(err)

				ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

				devPodUpOutput, _, err := f.ExecCommandCapture(ctx, []string{"up", "--debug", "--ide", "none", tempDir})
				framework.ExpectNoError(err)

				workspace, err := f.FindWorkspace(ctx, tempDir)
				framework.ExpectNoError(err)

				ids, err := dockerHelper.FindContainer(ctx, []string{
					fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
					fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				})
				framework.ExpectNoError(err)
				gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")
				gomega.Expect(devPodUpOutput).NotTo(gomega.ContainSubstring("Defaulting to a blank string."))
			}, ginkgo.SpecTimeout(60*time.Second))

			ginkgo.Context("with lifecycle commands", func() {
				ginkgo.It("should start a new workspace and execute array based lifecycle commands", func(ctx context.Context) {
					tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-lifecycle-array")
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
						fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
						fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
					})
					framework.ExpectNoError(err)
					gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")

					initializeCommand, err := os.ReadFile(filepath.Join(tempDir, "initialize-command.out"))
					framework.ExpectNoError(err)
					gomega.Expect(initializeCommand).To(gomega.ContainSubstring("initializeCommand"))

					onCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/on-create-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(onCreateCommand).To(gomega.ContainSubstring("onCreateCommand"))

					updateContentCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/update-content-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(updateContentCommand).To(gomega.Equal("updateContentCommand"))

					postCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-create-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(postCreateCommand).To(gomega.Equal("postCreateCommand"))

					postStartCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-start-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(postStartCommand).To(gomega.Equal("postStartCommand"))

					postAttachCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-attach-command.out", projectName})
					framework.ExpectNoError(err)
					gomega.Expect(postAttachCommand).To(gomega.Equal("postAttachCommand"))
				}, ginkgo.SpecTimeout(60*time.Second))

				//ginkgo.FIt("should start a new workspace and execute object based lifecycle commands", func(ctx context.Context) {
				//	tempDir, err := framework.CopyToTempDir("tests/up/testdata/docker-compose-lifecycle-object")
				//	framework.ExpectNoError(err)
				//	ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)
				//
				//	f := framework.NewDefaultFramework(initialDir + "/bin")
				//	_ = f.DevPodProviderAdd(ctx, "docker"})
				//	err = f.DevPodProviderUse(context.Background(), "docker")
				//	framework.ExpectNoError(err)
				//
				//	err = f.DevPodUp(ctx, tempDir)
				//	framework.ExpectNoError(err)
				//
				//	// Check for docker-compose container running
				//	projectName := composeHelper.ToProjectName(filepath.Base(tempDir))
				//	ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), projectName)
				//
				//	ids, err := dockerHelper.FindContainer(ctx, []string{
				//		fmt.Sprintf("%s=%s", compose.ProjectLabel, composeHelper.GetProjectName(workspace.UID)),
				//		fmt.Sprintf("%s=%s", compose.ServiceLabel, "app"),
				//	})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(ids).To(gomega.HaveLen(1), "1 compose container to be created")
				//
				//	initializeCommand, err := os.ReadFile(filepath.Join(tempDir, "initialize-command.out"))
				//	framework.ExpectNoError(err)
				//	gomega.Expect(initializeCommand).To(gomega.ContainSubstring("initializeCommand"))
				//
				//	onCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/on-create-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(onCreateCommand).To(gomega.ContainSubstring("onCreateCommand"))
				//
				//	updateContentCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/update-content-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(updateContentCommand).To(gomega.Equal("updateContentCommand"))
				//
				//	postCreateCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-create-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(postCreateCommand).To(gomega.Equal("postCreateCommand"))
				//
				//	postStartCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-start-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(postStartCommand).To(gomega.Equal("postStartCommand"))
				//
				//	postAttachCommand, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat $HOME/post-attach-command.out", projectName})
				//	framework.ExpectNoError(err)
				//	gomega.Expect(postAttachCommand).To(gomega.Equal("postAttachCommand"))
				//}, ginkgo.SpecTimeout(60*time.Second))
			})

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
