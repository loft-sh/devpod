GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Build the CLI and Desktop
.PHONY: build
build:
	BUILD_PLATFORMS=$(GOOS) ./hack/rebuild.sh

# Copy the devpod binary to the platform pod
.PHONY: cp-to-platform
cp-to-platform:
	SKIP_INSTALL=true BUILD_PLATFORMS=linux ./hack/rebuild.sh
	POD=$$(kubectl get pod -n loft -l app=loft,release=loft -o jsonpath='{.items[0].metadata.name}'); \
	echo "Copying ./test/devpod-linux-$(GOARCH) to pod $$POD"; \
	kubectl cp -n loft ./test/devpod-linux-$(GOARCH) $$POD:/usr/local/bin/devpod 
