package e2e

import (
	"testing"

	"github.com/onsi/ginkgo/v2"

	"github.com/onsi/gomega"

	// Register tests
	_ "github.com/loft-sh/devpod/e2e/tests/build"
	_ "github.com/loft-sh/devpod/e2e/tests/ide"
	_ "github.com/loft-sh/devpod/e2e/tests/integration"
	_ "github.com/loft-sh/devpod/e2e/tests/machine"
	_ "github.com/loft-sh/devpod/e2e/tests/machineprovider"
	_ "github.com/loft-sh/devpod/e2e/tests/provider"
	_ "github.com/loft-sh/devpod/e2e/tests/proxyprovider"
	_ "github.com/loft-sh/devpod/e2e/tests/ssh"
	_ "github.com/loft-sh/devpod/e2e/tests/up"
)

// TestRunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func TestRunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "DevPod e2e suite")
}
