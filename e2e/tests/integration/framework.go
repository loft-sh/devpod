package integration

import "github.com/onsi/ginkgo/v2"

// DevPodDescribe annotates the test with the label.
func DevPodDescribe(text string, body func()) bool {
	return ginkgo.Describe("[integration] "+text, body)
}
