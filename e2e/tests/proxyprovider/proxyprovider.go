package proxyprovider

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod proxy provider test suite", func() {
	ginkgo.Context("testing proxy providers", ginkgo.Label("proxyprovider"), ginkgo.Ordered, func() {
		ctx := context.Background()
		var initialDir string
		var devPodDir string

		ginkgo.BeforeEach(func() {
			var err error
			initialDir, err = os.Getwd()
			framework.ExpectNoError(err)

			devPodDir, err = framework.CopyToTempDir("tests/proxyprovider/testdata/proxyprovider")
			framework.ExpectNoError(err)

			// add & remove provider
			f := framework.NewDefaultFramework(initialDir + "/bin")
			err = f.DevPodProviderAdd(ctx, "./proxy-provider.yaml", "-o", "LOCATION="+devPodDir)
			framework.ExpectNoError(err)
			err = f.DevPodProviderUse(ctx, "proxy-provider")
			framework.ExpectNoError(err)
		})

		ginkgo.AfterEach(func() {
			// run after each
			f := framework.NewDefaultFramework(initialDir + "/bin")
			_ = f.DevPodProviderDelete(ctx, "proxy-provider")

			// remove temp dir
			framework.CleanupTempDir(initialDir, devPodDir)
		})

		ginkgo.It("create workspace via proxy provider", func(ctx context.Context) {
			f := framework.NewDefaultFramework(initialDir + "/bin")

			// copy test dir
			tempDir, err := framework.CopyToTempDirWithoutChdir(initialDir + "/tests/proxyprovider/testdata/docker")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				_ = os.RemoveAll(tempDir)
			})

			// create docker provider
			err = f.DevPodProviderAdd(ctx, filepath.Join(tempDir, "custom-docker-provider.yaml"), "--devpod-home", devPodDir)
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, tempDir, "--debug")
			framework.ExpectNoError(err)

			// expect secret to not be there
			fileBytes, err := os.ReadFile(filepath.Join(devPodDir, "agent", "contexts", "default", "workspaces", filepath.Base(tempDir), "workspace.json"))
			framework.ExpectNoError(err)
			framework.ExpectEqual(strings.Contains(string(fileBytes), "my-secret-value"), false, "workspace.json shouldn't contain provider secret")

			// expect workspace
			_, err = f.FindWorkspace(ctx, tempDir)
			framework.ExpectNoError(err)

			// delete workspace
			err = f.DevPodWorkspaceDelete(ctx, tempDir)
			framework.ExpectNoError(err)
		}, ginkgo.SpecTimeout(framework.GetTimeout()*2))

		ginkgo.It("create & stop workspace via proxy provider", func(ctx context.Context) {
			f := framework.NewDefaultFramework(initialDir + "/bin")

			// copy test dir
			tempDir, err := framework.CopyToTempDirWithoutChdir(initialDir + "/tests/proxyprovider/testdata/docker")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				_ = os.RemoveAll(tempDir)
			})

			// create docker provider
			err = f.DevPodProviderAdd(ctx, "docker", "--devpod-home", devPodDir)
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, tempDir, "--debug")
			framework.ExpectNoError(err)

			// expect workspace
			_, err = f.FindWorkspace(ctx, tempDir)
			framework.ExpectNoError(err)

			// check if ssh works
			err = f.DevPodSSHEchoTestString(ctx, tempDir)
			framework.ExpectNoError(err)

			// check if stop works
			err = f.DevPodStop(ctx, tempDir)
			framework.ExpectNoError(err)

			// check if status is stopped
			status, err := f.DevPodStatus(ctx, tempDir)
			framework.ExpectNoError(err)
			framework.ExpectEqual(status.State, client.StatusStopped, "state does not match")

			// delete workspace
			err = f.DevPodWorkspaceDelete(ctx, tempDir)
			framework.ExpectNoError(err)
		}, ginkgo.SpecTimeout(framework.GetTimeout()*2))

		ginkgo.It("recreate workspace", func(ctx context.Context) {
			f := framework.NewDefaultFramework(initialDir + "/bin")

			// copy test dir
			tempDir, err := framework.CopyToTempDirWithoutChdir(initialDir + "/tests/proxyprovider/testdata/docker-recreate")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				_ = os.RemoveAll(tempDir)
			})

			// create docker provider
			err = f.DevPodProviderAdd(ctx, "docker", "--devpod-home", devPodDir)
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, tempDir, "--debug")
			framework.ExpectNoError(err)

			// expect workspace
			_, err = f.FindWorkspace(ctx, tempDir)
			framework.ExpectNoError(err)

			// check if ssh works
			err = f.DevPodSSHEchoTestString(ctx, tempDir)
			framework.ExpectNoError(err)

			// delete & move .devcontainer.json
			err = os.Remove(filepath.Join(tempDir, ".devcontainer.json"))
			framework.ExpectNoError(err)
			err = os.Rename(filepath.Join(tempDir, ".devcontainer.json2"), filepath.Join(tempDir, ".devcontainer.json"))
			framework.ExpectNoError(err)

			// check if recreate works
			err = f.DevPodUp(ctx, tempDir, "--recreate")
			framework.ExpectNoError(err)

			// delete workspace
			err = f.DevPodWorkspaceDelete(ctx, tempDir)
			framework.ExpectNoError(err)
		}, ginkgo.SpecTimeout(framework.GetTimeout()*2))

		ginkgo.It("devcontainer path workspace", func(ctx context.Context) {
			f := framework.NewDefaultFramework(initialDir + "/bin")

			// copy test dir
			tempDir, err := framework.CopyToTempDirWithoutChdir(initialDir + "/tests/proxyprovider/testdata/docker-recreate")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(func() {
				_ = os.RemoveAll(tempDir)
			})

			// create docker provider
			err = f.DevPodProviderAdd(ctx, "docker", "--devpod-home", devPodDir)
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// wait for devpod workspace to come online (deadline: 30s)
			err = f.DevPodUp(ctx, tempDir, "--debug", "--devcontainer-path", ".devcontainer.json2")
			framework.ExpectNoError(err)

			// expect workspace
			_, err = f.FindWorkspace(ctx, tempDir)
			framework.ExpectNoError(err)

			// delete workspace
			err = f.DevPodWorkspaceDelete(ctx, tempDir)
			framework.ExpectNoError(err)
		}, ginkgo.SpecTimeout(framework.GetTimeout()*2))
	})
})
