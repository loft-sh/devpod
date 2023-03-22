package e2e

import (
	"math/rand"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/onsi/gomega"

	// Register tests
	_ "github.com/loft-sh/devpod/e2e/tests/ssh"
	_ "github.com/loft-sh/devpod/e2e/tests/up"
)

// TestRunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func TestRunE2ETests(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "DevPod e2e suite")
}
