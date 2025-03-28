GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
SKIP_INSTALL := false
GINKGO_LABEL_FILTER ?=

# Platform host
PLATFORM_HOST := localhost:8080

# Tests config
KIND_CLUSTER_NAME := devpod-e2e
GOLANGCILINT_CONFIG := .golangci.yaml

.PHONY: help
help: ## Show this help.
	@echo "Available targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'


.PHONY: build
build: ## Build the CLI and Desktop
	SKIP_INSTALL=$(SKIP_INSTALL) BUILD_PLATFORMS=$(GOOS) BUILD_ARCHS=$(GOARCH) ./hack/rebuild.sh

.PHONY: run-desktop
run-desktop: build ## Run the desktop app
	cd desktop && yarn desktop:dev


.PHONY: run-daemon
run-daemon: build ## Run the daemon against loft host
	devpod pro daemon start --host $(PLATFORM_HOST)

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run -c $(GOLANGCILINT_CONFIG)

# Namespace to use for the platform
NAMESPACE := loft

.PHONY: cp-to-platform
cp-to-platform: ## Copy the devpod binary to the platform pod
	SKIP_INSTALL=true BUILD_PLATFORMS=linux BUILD_ARCHS=$(GOARCH) ./hack/rebuild.sh
	POD=$$(kubectl get pod -n $(NAMESPACE) -l app=loft,release=loft -o jsonpath='{.items[0].metadata.name}'); \
	echo "Copying ./test/devpod-linux-$(GOARCH) to pod $$POD"; \
	kubectl cp -n $(NAMESPACE) ./test/devpod-linux-$(GOARCH) $$POD:/usr/local/bin/devpod

.PHONY: kind
kind: ## Create kind cluster for e2e tests
	kind create cluster --name $(KIND_CLUSTER_NAME)

.PHONY: build-e2e
build-e2e: ## Build bin for e2e tests
	SKIP_INSTALL=$(SKIP_INSTALL) BUILD_PLATFORMS=$(GOOS) BUILD_ARCHS=$(GOARCH) BUILDDIR=e2e/bin ./hack/rebuild.sh


.PHONY: e2e
e2e: ## Run e2e tests
	@cd e2e && go test -v -ginkgo.v -timeout 3600s --ginkgo.label-filter="$(GINKGO_LABEL_FILTER)" ./...

.PHONY: e2e-up
e2e-up:
	@$(MAKE) e2e GINKGO_LABEL_FILTER=up
