package machine

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod machine create", func() {
	ginkgo.Context("testing machine", ginkgo.Label("machine"), ginkgo.Ordered, func() {
		ctx := context.Background()
		initialDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		ginkgo.It("should add simple machine and then delete it", func() {
			tempDir, err := framework.CopyToTempDir("tests/machine/testdata")
			framework.ExpectNoError(err)
			ginkgo.DeferCleanup(framework.CleanupTempDir, initialDir, tempDir)

			f := framework.NewDefaultFramework(initialDir + "/bin")

			// Ensure that mock-provider is deleted
			_ = f.DevPodProviderDelete(ctx, "mock-provider")

			ginkgo.By("Add mock provider")
			err = f.DevPodProviderAdd(ctx, tempDir+"/mock-provider.yaml")
			framework.ExpectNoError(err)

			ginkgo.By("Use mock provier")
			err = f.DevPodProviderUse(context.Background(), "mock-provider")
			framework.ExpectNoError(err)

			machineUUID, _ := uuid.NewRandom()
			machineName := machineUUID.String()

			ginkgo.By("Create test machine with mock provider")
			err = f.DevPodMachineCreate([]string{machineName})
			framework.ExpectNoError(err)

			ginkgo.By("Remove test machine")
			err = f.DevPodMachineDelete([]string{machineName})
			framework.ExpectNoError(err)
		})
	})
})
