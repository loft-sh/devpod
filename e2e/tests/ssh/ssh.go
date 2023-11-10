package ssh

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod ssh test suite", func() {
	ctx := context.Background()
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ginkgo.It("should start a new workspace with a docker provider (default) and ssh into it", func() {
		tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/local-test")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")
		_ = f.DevPodProviderAdd(ctx, "docker")
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

		// Start up devpod workspace
		devpodUpDeadline := time.Now().Add(5 * time.Minute)
		devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
		defer cancel()
		err = f.DevPodUp(devpodUpCtx, tempDir)
		framework.ExpectNoError(err)

		devpodSSHDeadline := time.Now().Add(20 * time.Second)
		devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
		defer cancelSSH()
		err = f.DevPodSSHEchoTestString(devpodSSHCtx, tempDir)
		framework.ExpectNoError(err)
	})

	ginkgo.It("should start a new workspace with a docker provider (default) and forward gpg agent into it", func() {
		tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/gpg-forwarding")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")
		_ = f.DevPodProviderAdd(ctx, "docker")
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

		out, err := exec.Command("gpg", "-k").Output()
		if err != nil || len(out) == 0 {
			err = f.SetupGPG(tempDir)
			framework.ExpectNoError(err)
		}

		// Start up devpod workspace
		devpodUpDeadline := time.Now().Add(5 * time.Minute)
		devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
		defer cancel()
		err = f.DevPodUp(devpodUpCtx, tempDir, "--gpg-agent-forwarding")
		framework.ExpectNoError(err)

		devpodSSHDeadline := time.Now().Add(20 * time.Second)
		devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
		defer cancelSSH()
		err = f.DevPodSSHGpgTestKey(devpodSSHCtx, tempDir)
		framework.ExpectNoError(err)
	})

	ginkgo.It("should start a new workspace with a docker provider (default) and forward a port into it", func() {
		tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/forward-test")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")
		_ = f.DevPodProviderAdd(ctx, "docker")
		err = f.DevPodProviderUse(context.Background(), "docker")
		framework.ExpectNoError(err)

		ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

		// Create a new random number generator with a custom seed (e.g., current time)
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)

		// Start up devpod workspace
		devpodUpDeadline := time.Now().Add(5 * time.Minute)
		devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
		defer cancel()
		err = f.DevPodUp(devpodUpCtx, tempDir)
		framework.ExpectNoError(err)

		// Generate a random number for the server port between 50000 and 51000
		port := rng.Intn(1000) + 50000

		fmt.Println("Running netcat server on port", port)

		devpodSSHDeadline := time.Now().Add(20 * time.Second)
		devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
		defer cancelSSH()
		// start ssh with netcat server in background
		go func() {
			_ = f.DevpodPortTest(devpodSSHCtx, strconv.Itoa(port), tempDir)
		}()

		// wait 5s just to be sure the netcat server has time to start
		time.Sleep(time.Second * 5)

		err = exec.Command("nc", "-zv", "localhost", strconv.Itoa(port)).Run()
		if err == nil {
			fmt.Println("forwarding successful")
		}
		framework.ExpectNoError(err)
	})
})
