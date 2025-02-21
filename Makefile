GOOS := $(shell go env GOOS)

# Build the CLI and Desktop
.PHONY: build
build:
	BUILD_PLATFORMS=$(GOOS) ./hack/rebuild.sh
