GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
SKIP_INSTALL := false

# Platform host
PLATFORM_HOST := localhost:8080

# Build the CLI and Desktop
.PHONY: build
build:
	SKIP_INSTALL=$(SKIP_INSTALL) BUILD_PLATFORMS=$(GOOS) BUILD_ARCHS=$(GOARCH) ./hack/rebuild.sh

# Run the desktop app
.PHONY: run-desktop
run-desktop: build
	cd desktop && yarn desktop:dev:debug

# Run the daemon against loft host
.PHONY: run-daemon
run-daemon: build
	devpod pro daemon start --host $(PLATFORM_HOST)

# Namespace to use for the platform
NAMESPACE := loft

# Copy the devpod binary to the platform pod
.PHONY: cp-to-platform
cp-to-platform:
	SKIP_INSTALL=true BUILD_PLATFORMS=linux BUILD_ARCHS=$(GOARCH) ./hack/rebuild.sh
	POD=$$(kubectl get pod -n $(NAMESPACE) -l app=loft,release=loft -o jsonpath='{.items[0].metadata.name}'); \
	echo "Copying ./test/devpod-linux-$(GOARCH) to pod $$POD"; \
	kubectl cp -n $(NAMESPACE) ./test/devpod-linux-$(GOARCH) $$POD:/usr/local/bin/devpod 
