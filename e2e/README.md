### E2E tests

#### Prerequisites

Make sure you have ginkgo installed on your local machine:
```
go get github.com/onsi/ginkgo/ginkgo
```

To build the binaries locally use the following command from this directory
```
BUILDDIR=bin SRCDIR=".." ../hack/build-e2e.sh
```

#### Run all E2E test
```
# Install ginkgo and run in this directory
ginkgo
```
