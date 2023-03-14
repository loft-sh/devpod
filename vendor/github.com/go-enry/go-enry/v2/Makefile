LINGUIST_PATH = .linguist

# shared objects
RESOURCES_DIR=./.shared
LINUX_DIR=$(RESOURCES_DIR)/linux-x86-64
LINUX_SHARED_LIB=$(LINUX_DIR)/libenry.so
DARWIN_DIR=$(RESOURCES_DIR)/darwin
DARWIN_SHARED_LIB=$(DARWIN_DIR)/libenry.dylib
STATIC_LIB=$(RESOURCES_DIR)/libenry.a
HEADER_FILE=libenry.h
NATIVE_LIB=./shared/enry.go

$(LINGUIST_PATH):
	git clone https://github.com/github/linguist.git $@

clean-linguist:
	rm -rf $(LINGUIST_PATH)

clean-shared:
	rm -rf $(RESOURCES_DIR)

clean: clean-linguist clean-shared

code-generate: $(LINGUIST_PATH)
	mkdir -p data && \
	go run internal/code-generator/main.go
	ENRY_TEST_REPO="$${PWD}/.linguist" go test  -v \
		-run Test_GeneratorTestSuite \
		./internal/code-generator/generator \
		-testify.m TestUpdateGeneratorTestSuiteGold \
		-update_gold

benchmarks: $(LINGUIST_PATH)
	go test -run=NONE -bench=. && \
	benchmarks/linguist-total.rb

benchmarks-samples: $(LINGUIST_PATH)
	go test -run=NONE -bench=. -benchtime=5us && \
	benchmarks/linguist-samples.rb

benchmarks-slow: $(LINGUIST_PATH)
	mkdir -p benchmarks/output && \
	go test -run=NONE -bench=. -slow -benchtime=100ms -timeout=100h > benchmarks/output/enry_samples.bench && \
	benchmarks/linguist-samples.rb 5 > benchmarks/output/linguist_samples.bench

linux-shared: $(LINUX_SHARED_LIB)

darwin-shared: $(DARWIN_SHARED_LIB)

$(DARWIN_SHARED_LIB):
	mkdir -p $(DARWIN_DIR) && \
	CC="o64-clang" CXX="o64-clang++" CGO_ENABLED=1 GOOS=darwin go build -buildmode=c-shared -o $(DARWIN_SHARED_LIB) $(NATIVE_LIB) && \
	mv $(DARWIN_DIR)/$(HEADER_FILE) $(RESOURCES_DIR)/$(HEADER_FILE)

$(LINUX_SHARED_LIB):
	mkdir -p $(LINUX_DIR) && \
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o $(LINUX_SHARED_LIB) $(NATIVE_LIB) && \
	mv $(LINUX_DIR)/$(HEADER_FILE) $(RESOURCES_DIR)/$(HEADER_FILE)


static: $(STATIC_LIB)

$(STATIC_LIB):
	CGO_ENABLED=1 go build -buildmode=c-archive -o $(STATIC_LIB) $(NATIVE_LIB)

.PHONY: benchmarks benchmarks-samples benchmarks-slow
