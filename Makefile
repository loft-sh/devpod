GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Platform host
PLATFORM_HOST := localhost:8080

# Build the CLI and Desktop
.PHONY: build
build:
	BUILD_PLATFORMS=$(GOOS) ./hack/rebuild.sh

# Run the desktop app
.PHONY: run-desktop
run-desktop: build
	cd desktop && yarn desktop:dev

# Run the daemon against loft host
.PHONY: run-daemon
run-daemon: build
	devpod pro daemon start --host $(PLATFORM_HOST) 

# Copy the devpod binary to the platform pod
.PHONY: cp-to-platform
cp-to-platform:
	SKIP_INSTALL=true BUILD_PLATFORMS=linux ./hack/rebuild.sh
	POD=$$(kubectl get pod -n loft -l app=loft,release=loft -o jsonpath='{.items[0].metadata.name}'); \
	echo "Copying ./test/devpod-linux-$(GOARCH) to pod $$POD"; \
	kubectl cp -n loft ./test/devpod-linux-$(GOARCH) $$POD:/usr/local/bin/devpod 
