package machineprovider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod machine provider test suite", func() {
	var initialDir string

	ginkgo.BeforeEach(func() {
		var err error
		initialDir, err = os.Getwd()
		framework.ExpectNoError(err)
	})

	ginkgo.It("test start / stop / status", func(ctx context.Context) {
		f := framework.NewDefaultFramework(initialDir + "/bin")

		// copy test dir
		tempDir, err := framework.CopyToTempDirWithoutChdir(initialDir + "/tests/machineprovider/testdata/machineprovider")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			_ = os.RemoveAll(tempDir)
		})

		tempDirLocation, err := os.MkdirTemp("", "")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			_ = os.RemoveAll(tempDirLocation)
		})

		// create docker provider
		err = f.DevPodProviderAdd(ctx, filepath.Join(tempDir, "provider.yaml"), "-o", "LOCATION="+tempDirLocation)
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			err = f.DevPodProviderDelete(context.Background(), "docker123")
			framework.ExpectNoError(err)
		})
		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

		// wait for devpod workspace to come online (deadline: 30s)
		err = f.DevPodUp(ctx, tempDir, "--debug")
		framework.ExpectNoError(err)

		// expect workspace
		workspace, err := f.FindWorkspace(ctx, tempDir)
		framework.ExpectNoError(err)

		// check status
		status, err := f.DevPodStatus(ctx, tempDir)
		framework.ExpectNoError(err)
		framework.ExpectEqual(strings.ToUpper(status.State), "RUNNING", "workspace status did not match")

		// stop container
		err = f.DevPodStop(ctx, tempDir)
		framework.ExpectNoError(err)

		// check status
		status, err = f.DevPodStatus(ctx, tempDir)
		framework.ExpectNoError(err)
		framework.ExpectEqual(strings.ToUpper(status.State), "STOPPED", "workspace status did not match")

		// wait for devpod workspace to come online (deadline: 30s)
		err = f.DevPodUp(ctx, tempDir)
		framework.ExpectNoError(err)

		// check if ssh works as it should start the container
		out, err := f.DevPodSSH(ctx, tempDir, fmt.Sprintf("cat /workspaces/%s/test.txt", workspace.ID))
		framework.ExpectNoError(err)
		framework.ExpectEqual(out, "Test123", "workspace content does not match")

		// delete workspace
		err = f.DevPodWorkspaceDelete(ctx, tempDir)
		framework.ExpectNoError(err)
	}, ginkgo.SpecTimeout(60*time.Second))

	ginkgo.It("test devpod inactivity timeout", func(ctx context.Context) {
		f := framework.NewDefaultFramework(initialDir + "/bin")

		// copy test dir
		tempDir, err := framework.CopyToTempDirWithoutChdir(initialDir + "/tests/machineprovider/testdata/machineprovider2")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			err = os.RemoveAll(tempDir)
			framework.ExpectNoError(err)
		})

		tempDirLocation, err := os.MkdirTemp("", "")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			err = os.RemoveAll(tempDirLocation)
			framework.ExpectNoError(err)
		})

		// create provider
		_ = f.DevPodProviderDelete(ctx, "docker123")
		err = f.DevPodProviderAdd(ctx, filepath.Join(tempDir, "provider.yaml"))
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			err = f.DevPodProviderDelete(context.Background(), "docker123")
			framework.ExpectNoError(err)
		})
		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

		// wait for devpod workspace to come online (deadline: 30s)
		err = f.DevPodUp(ctx, tempDir, "--debug", "--daemon-interval=3s")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(func() {
			// delete workspace
			err = f.DevPodWorkspaceDelete(context.Background(), tempDir)
			framework.ExpectNoError(err)
		})

		// check status
		status, err := f.DevPodStatus(ctx, tempDir, "--container-status=false")
		framework.ExpectNoError(err)
		framework.ExpectEqual(strings.ToUpper(status.State), "RUNNING", "workspace status did not match")

		// stop container
		err = f.DevPodStop(ctx, tempDir)
		framework.ExpectNoError(err)

		// check status
		status, err = f.DevPodStatus(ctx, tempDir, "--container-status=false")
		framework.ExpectNoError(err)
		framework.ExpectEqual(strings.ToUpper(status.State), "STOPPED", "workspace status did not match")

		// wait for devpod workspace to come online (deadline: 30s)
		err = f.DevPodUp(ctx, tempDir, "--daemon-interval=3s")
		framework.ExpectNoError(err)

		// check status
		status, err = f.DevPodStatus(ctx, tempDir, "--container-status=false")
		framework.ExpectNoError(err)
		framework.ExpectEqual(strings.ToUpper(status.State), "RUNNING", "workspace status did not match")

		// wait until workspace is stopped again
		now := time.Now()
		for {
			status, err := f.DevPodStatus(ctx, tempDir, "--container-status=false")
			framework.ExpectNoError(err)
			framework.ExpectEqual(time.Since(now) < time.Minute*2, true, "machine did not shutdown in time")
			if status.State == "Stopped" {
				break
			}

			time.Sleep(time.Second * 2)
		}
	}, ginkgo.SpecTimeout(300*time.Second))
})
