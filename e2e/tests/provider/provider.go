package provider

import (
	"context"
	"os"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod provider test suite", func() {
	ctx := context.Background()
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	ginkgo.It("should add simple provider and delete it", func() {
		tempDir, err := framework.CopyToTempDir("tests/provider/testdata/simple-k8s-provider")
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")

		// Ensure that provider 1 is deleted
		err = f.DevPodProviderDelete(ctx, "provider1", "--ignore-not-found")
		framework.ExpectNoError(err)

		// Add provider 1
		err = f.DevPodProviderAdd(ctx, tempDir+"/provider1.yaml")
		framework.ExpectNoError(err)

		// Ensure provider 1 exists but not provider X
		err = f.DevPodProviderUse(context.Background(), "provider1")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "providerX")
		framework.ExpectError(err)

		// Cleanup: delete provider 1
		err = f.DevPodProviderDelete(ctx, "provider1")
		framework.ExpectNoError(err)

		// Cleanup: ensure provider 1 is deleted
		err = f.DevPodProviderUse(context.Background(), "provider1")
		framework.ExpectError(err)
	})

	ginkgo.It("should add simple provider and update it", func() {
		tempDir, err := framework.CopyToTempDir("tests/provider/testdata/simple-k8s-provider")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		f := framework.NewDefaultFramework(initialDir + "/bin")

		// Ensure that provider 2 is deleted
		err = f.DevPodProviderDelete(ctx, "provider2", "--ignore-not-found")
		framework.ExpectNoError(err)

		// Add provider 2 and use it
		err = f.DevPodProviderAdd(ctx, tempDir+"/provider2.yaml")
		framework.ExpectNoError(err)
		err = f.DevPodProviderUse(context.Background(), "provider2")
		framework.ExpectNoError(err)

		// Ensure provider 2 namespace parameter has the default value
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
		err = f.DevPodProviderOptionsCheckNamespaceDescription(ctx, "provider2", "The namespace to use")
		framework.ExpectNoError(err)
		cancel()

		// Update provider 2 (change the namespace description value)
		err = f.DevPodProviderUpdate(context.Background(), "provider2", tempDir+"/provider2-update.yaml")
		framework.ExpectNoError(err)

		// Ensure that provider 2 was updated
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
		err = f.DevPodProviderOptionsCheckNamespaceDescription(ctx, "provider2", "Updated namespace parameter")
		framework.ExpectNoError(err)
		cancel()

		// Cleanup: delete provider 2
		err = f.DevPodProviderDelete(context.Background(), "provider2")
		framework.ExpectNoError(err)

		// Cleanup: ensure provider 2 is deleted
		err = f.DevPodProviderUse(context.Background(), "provider2")
		framework.ExpectError(err)
	})
})
