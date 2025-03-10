#!/usr/bin/env make

all:	build

build:	deps lint vet

clean:
	(cd test; $(MAKE) clean)

test:	tests
tests:	build
	(cd test && $(MAKE) tests)


#
#  Update dependencies.
#
deps:
	@echo "  - Update dependencies ..."
	go mod tidy

	@echo "  - Download go modules ..."
	go mod download   #  -x


#
#  Lint and vet targets.
#
lint:
	(cd test && $(MAKE) lint)

	@echo  "  - Linting README ..."
	@(command -v mdl > /dev/null && mdl README.md ||  \
	    echo "Warning: mdl command not found - skipping README.md lint ...")

	@echo  "  - Linting sources ..."
	gofmt -d -s reaper.go
	@echo  "  - Linter checks passed."


vet:
	(cd test && $(MAKE) vet)

	@echo  "  - Vetting go sources ..."
	go vet ./...
	@echo  "  - go vet checks passed."


.PHONY:	build clean test tests deps lint vet
