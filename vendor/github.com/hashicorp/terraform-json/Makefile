GOTOOLS = \
	gotest.tools/gotestsum@latest

test: tools
	gotestsum --format=short-verbose $(TEST) $(TESTARGS)

generate:
	cd testdata && make generate

modules:
	go mod download && go mod verify

test-circle:
	mkdir -p test-results/terraform-json
	gotestsum --format=short-verbose --junitfile test-results/terraform-json/results.xml

tools:
	@echo $(GOTOOLS) | xargs -t -n1 go install
	go mod tidy

.PHONY: test generate modules test-circle tools
