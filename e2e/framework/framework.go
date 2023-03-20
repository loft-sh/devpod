package framework

import (
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
)

type Framework struct {
	TestDirectory string
}

func (f *Framework) SetupTestDirectory() error {
	dir, err := os.MkdirTemp("devpod-e2e", "test")
	if err != nil {
		return err
	}
	f.TestDirectory = dir
	return nil
}

func (f *Framework) TeardownTestDirectory() error {
	return os.RemoveAll(f.TestDirectory)
}

func RegisterTestCase(testsuite, testcase string, fn func()) bool {
	return ginkgo.Describe(fmt.Sprintf("[%s]: %s", testsuite, testcase), fn)
}
