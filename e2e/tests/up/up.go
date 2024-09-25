package up

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/e2e/framework"
	config2 "github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	docker "github.com/loft-sh/devpod/pkg/docker"
	"github.com/loft-sh/devpod/pkg/language"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = DevPodDescribe("devpod up test suite", func() {
	ginkgo.Context("testing up command", ginkgo.Label("up"), ginkgo.Ordered, func() {
		var dockerHelper *docker.DockerHelper
		var initialDir string

		ginkgo.BeforeEach(func() {
			var err error
			initialDir, err = os.Getwd()
			framework.ExpectNoError(err)

			dockerHelper = &docker.DockerHelper{DockerCommand: "docker"}
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
			err = f.DevPodUp(ctx, "https://github.com/microsoft/vscode-remote-try-python.git")
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

			// set env vars with file
			tmpDir, err := framework.CreateTempDir()
			framework.ExpectNoError(err)

			// create invalid env file
			invalidData := []byte("TEST VAR=" + value)
			workspaceEnvFileInvalid := filepath.Join(tmpDir, ".invalid")
			err = os.WriteFile(
				workspaceEnvFileInvalid,
				invalidData, 0o644)
			framework.ExpectNoError(err)
			defer os.Remove(workspaceEnvFileInvalid)

			// set env var
			err = f.DevPodUp(ctx, name, "--workspace-env-file", workspaceEnvFileInvalid)
			framework.ExpectError(err)

			// create valid env file
			validData := []byte("TEST_VAR=" + value)
			workspaceEnvFileValid := filepath.Join(tmpDir, ".valid")
			err = os.WriteFile(
				workspaceEnvFileValid,
				validData, 0o644)
			framework.ExpectNoError(err)
			defer os.Remove(workspaceEnvFileValid)

			// set env var
			err = f.DevPodUp(ctx, name, "--workspace-env-file", workspaceEnvFileValid)
			framework.ExpectNoError(err)

			// check env var
			out, err = f.DevPodSSH(ctx, name, "echo -n $TEST_VAR")
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, value, "should be set now")

			// delete env var
			err = f.DevPodUp(ctx, name, "--workspace-env", "TEST_VAR=")
			framework.ExpectNoError(err)

			// check env var
			out, err = f.DevPodSSH(ctx, name, "echo -n $TEST_VAR")
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, "", "should be empty")

			// create a second valid env file with a different env var
			validData = []byte("TEST_OTHER_VAR=" + value)
			workspaceEnvFileValid2 := filepath.Join(tmpDir, ".valid2")
			err = os.WriteFile(
				workspaceEnvFileValid2,
				validData, 0o644)
			framework.ExpectNoError(err)
			defer os.Remove(workspaceEnvFileValid2)

			// set env var from both files
			err = f.DevPodUp(ctx, name, "--workspace-env-file", fmt.Sprintf("%s,%s", workspaceEnvFileValid, workspaceEnvFileValid2))
			framework.ExpectNoError(err)

			// check env var from .valid file
			out, err = f.DevPodSSH(ctx, name, "echo -n $TEST_VAR")
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, value, "should be set now")

			// check env var from .valid2 file
			out, err = f.DevPodSSH(ctx, name, "echo -n $TEST_OTHER_VAR")
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, value, "should be set now")
		})

		ginkgo.It("should allow checkout of a GitRepo from a commit hash", func() {
			ctx := context.Background()
			f := framework.NewDefaultFramework(initialDir + "/bin")

			_ = f.DevPodProviderDelete(ctx, "docker")
			err := f.DevPodProviderAdd(ctx, "docker")
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, "docker")
			framework.ExpectNoError(err)

			name := "sha256-0c1547c"
			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), name)

			// Wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, "github.com/microsoft/vscode-remote-try-python@sha256:0c1547c")
			framework.ExpectNoError(err)
		})

		ginkgo.It("should allow checkout of a GitRepo from a pull request reference", func() {
			ctx := context.Background()
			f := framework.NewDefaultFramework(initialDir + "/bin")

			_ = f.DevPodProviderDelete(ctx, "docker")
			err := f.DevPodProviderAdd(ctx, "docker")
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, "docker")
			framework.ExpectNoError(err)

			name := "PR3"
			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), name)

			// Wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, "github.com/loft-sh/devpod@pull/3/head")
			framework.ExpectNoError(err)
		})

		ginkgo.It("should allow checkout of a private GitRepo", func() {
			// need to debug
			if runtime.GOOS == "windows" {
				ginkgo.Skip("skipping on windows")
			}

			username := os.Getenv("GH_USERNAME")
			token := os.Getenv("GH_ACCESS_TOKEN")

			if username == "" || token == "" {
				ginkgo.Skip("WARNING: skipping test, secrets not found")
			}

			ctx := context.Background()
			f := framework.NewDefaultFramework(initialDir + "/bin")

			_ = f.DevPodProviderDelete(ctx, "docker")
			err := f.DevPodProviderAdd(ctx, "docker")
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, "docker")
			framework.ExpectNoError(err)

			// setup git credentials
			err = exec.Command("git", []string{"config", "--global", "credential.helper", "store"}...).Run()
			framework.ExpectNoError(err)

			gitCredentialString := []byte("https://" + username + ":" + token + "@github.com")
			err = os.WriteFile(
				filepath.Join(os.Getenv("HOME"), ".git-credentials"),
				gitCredentialString, 0o644)
			framework.ExpectNoError(err)
			defer os.Remove(filepath.Join(os.Getenv("HOME"), ".git-credentials"))

			name := "testprivaterepo"
			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), name)

			// Wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, "https://github.com/"+username+"/test_private_repo.git")
			framework.ExpectNoError(err)

			// Ensure git credentials are properly forwarded by cloning the private repo
			// from within the container
			out, err := f.DevPodSSH(ctx, name, "git clone https://github.com/"+username+"/test_private_repo")
			framework.ExpectNoError(err)
			fmt.Println(out)
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
			list := &framework.PodList{}
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
			list = &framework.PodList{}
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
			list = &framework.PodList{}
			err = json.Unmarshal(stdout, list)
			framework.ExpectNoError(err)
			framework.ExpectEqual(len(list.Items), 1, "Expect 1 pod")

			// check if ssh works
			err = f.DevPodSSHEchoTestString(ctx, tempDir)
			framework.ExpectNoError(err)

			// export workspace
			data, err := f.ExecCommandOutput(ctx, []string{"export", tempDir})
			framework.ExpectNoError(err)

			// check if file is there
			out, err := os.ReadFile(filepath.Join(tempDir, "test_file.txt"))
			framework.ExpectNoError(err)
			framework.ExpectEqual(strings.TrimSpace(string(out)), "test")

			// delete devpod directory & temp dir
			configDir, err := config2.GetConfigDir()
			framework.ExpectNoError(err)
			err = os.RemoveAll(configDir)
			framework.ExpectNoError(err)
			err = os.RemoveAll(tempDir)
			framework.ExpectNoError(err)

			// import workspace
			_, err = f.ExecCommandOutput(ctx, []string{"import", "--data", data})
			framework.ExpectNoError(err)

			// check if ssh works
			err = f.DevPodSSHEchoTestString(ctx, tempDir)
			framework.ExpectNoError(err)

			// make sure file is not there anymore
			_, err = os.ReadFile(filepath.Join(tempDir, "test_file.txt"))
			framework.ExpectError(err)
			_, err = os.ReadFile(filepath.Join(tempDir, ".devcontainer.json"))
			framework.ExpectNoError(err)

			// run up
			err = f.DevPodUp(ctx, tempDir)
			framework.ExpectNoError(err)

			// check if ssh works
			err = f.DevPodSSHEchoTestString(ctx, tempDir)
			framework.ExpectNoError(err)

			// delete workspace
			err = f.DevPodWorkspaceDelete(ctx, tempDir)
			framework.ExpectNoError(err)
		})

		ginkgo.It("create workspace without devcontainer.json", func() {
			const providerName = "test-docker"
			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")
			tempDir, err := framework.CopyToTempDir("tests/up/testdata/no-devcontainer")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			// provider add, use and delete afterwards
			err = f.DevPodProviderAdd(ctx, "docker", "--name", providerName)
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, providerName)
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				err = f.DevPodProviderDelete(ctx, providerName)
				framework.ExpectNoError(err)
			})

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

			devcontainerPath := filepath.Join("/workspaces", projectName, ".devcontainer.json")

			containerEnvPath, _, err := f.ExecCommandCapture(ctx, []string{"ssh", "--command", "cat " + devcontainerPath, projectName})
			framework.ExpectNoError(err)
			expectedImageName := language.MapConfig[language.Go].ImageContainer.Image

			gomega.Expect(containerEnvPath).To(gomega.Equal(fmt.Sprintf("{\"image\":\"%s\"}", expectedImageName)))

			err = f.DevPodWorkspaceDelete(ctx, tempDir)
			framework.ExpectNoError(err)
		})

		ginkgo.It("recreate a local workspace", func() {
			const providerName = "test-docker"
			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")
			tempDir, err := framework.CopyToTempDir("tests/up/testdata/no-devcontainer")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			// provider add, use and delete afterwards
			err = f.DevPodProviderAdd(ctx, "docker", "--name", providerName)
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, providerName)
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				err = f.DevPodProviderDelete(ctx, providerName)
				framework.ExpectNoError(err)
			})

			err = f.DevPodUp(ctx, tempDir)
			framework.ExpectNoError(err)

			// recreate
			err = f.DevPodUpRecreate(ctx, tempDir)
			framework.ExpectNoError(err)

			err = f.DevPodWorkspaceDelete(ctx, tempDir)
			framework.ExpectNoError(err)
		})

		ginkgo.It("create workspace in a subpath", func() {
			const providerName = "test-docker"
			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")

			// provider add, use and delete afterwards
			err := f.DevPodProviderAdd(ctx, "docker", "--name", providerName)
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, providerName)
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				err = f.DevPodProviderDelete(ctx, providerName)
				framework.ExpectNoError(err)
			})

			err = f.DevPodUp(ctx, "https://github.com/loft-sh/examples@subpath:/devpod/jupyter-notebook-hello-world")
			framework.ExpectNoError(err)

			id := "subpath--devpod-jupyter-notebook-hello-world"
			out, err := f.DevPodSSH(ctx, id, "pwd")
			framework.ExpectNoError(err)
			framework.ExpectEqual(out, fmt.Sprintf("/workspaces/%s\n", id), "should be subpath")

			err = f.DevPodWorkspaceDelete(ctx, id)
			framework.ExpectNoError(err)
		})

		ginkgo.It("recreate a remote workspace", func() {
			const providerName = "test-docker"
			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")

			// provider add, use and delete afterwards
			err := f.DevPodProviderAdd(ctx, "docker", "--name", providerName)
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, providerName)
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				err = f.DevPodProviderDelete(ctx, providerName)
				framework.ExpectNoError(err)
			})

			id := "subpath--devpod-jupyter-notebook-hello-world"
			err = f.DevPodUp(ctx, "https://github.com/loft-sh/examples@subpath:/devpod/jupyter-notebook-hello-world")
			framework.ExpectNoError(err)

			_, err = f.DevPodSSH(ctx, id, "pwd")
			framework.ExpectNoError(err)

			// recreate
			err = f.DevPodUpRecreate(ctx, "https://github.com/loft-sh/examples@subpath:/devpod/jupyter-notebook-hello-world")
			framework.ExpectNoError(err)

			_, err = f.DevPodSSH(ctx, id, "pwd")
			framework.ExpectNoError(err)

			err = f.DevPodWorkspaceDelete(ctx, id)
			framework.ExpectNoError(err)
		})

		ginkgo.It("reset a remote workspace", func() {
			const providerName = "test-docker"
			ctx := context.Background()

			f := framework.NewDefaultFramework(initialDir + "/bin")

			// provider add, use and delete afterwards
			err := f.DevPodProviderAdd(ctx, "docker", "--name", providerName)
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, providerName)
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				err = f.DevPodWorkspaceDelete(ctx, "jupyter-notebook-hello-world")
				framework.ExpectNoError(err)
				err = f.DevPodProviderDelete(ctx, providerName)
				framework.ExpectNoError(err)
			})

			id := "subpath--devpod-jupyter-notebook-hello-world"
			err = f.DevPodUp(ctx, "https://github.com/loft-sh/examples@subpath:/devpod/jupyter-notebook-hello-world")
			framework.ExpectNoError(err)

			// create files in root and in workspace, after create we expect data to still be there
			_, err = f.DevPodSSH(ctx, id, fmt.Sprintf("sudo touch /workspaces/%s/DATA", id))
			framework.ExpectNoError(err)
			_, err = f.DevPodSSH(ctx, id, "sudo touch /ROOTFS")
			framework.ExpectNoError(err)

			// reset
			err = f.DevPodUpReset(ctx, "https://github.com/loft-sh/examples/@subpath:/devpod/jupyter-notebook-hello-world")
			framework.ExpectNoError(err)

			// this should fail! because --reset should trigger a new git clone
			_, err = f.DevPodSSH(ctx, id, fmt.Sprintf("ls /workspaces/%s/DATA", id))
			framework.ExpectError(err)
			// this should fail! because --recreare should trigger a new build, so a new rootfs
			_, err = f.DevPodSSH(ctx, id, "ls /ROOTFS")
			framework.ExpectError(err)

			err = f.DevPodWorkspaceDelete(ctx, id)
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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
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
			}, ginkgo.SpecTimeout(framework.GetTimeout()))
		})
	})
})
