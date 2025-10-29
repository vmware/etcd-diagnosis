TESTFLAGS_RACE=-race=false
CGO_ENABLED=0
ifdef ENABLE_RACE
	TESTFLAGS_RACE=-race=true
	CGO_ENABLED=1
endif

TESTFLAGS_CPU=
ifdef CPU
	TESTFLAGS_CPU=-cpu=$(CPU)
endif
TESTFLAGS = $(TESTFLAGS_RACE) $(TESTFLAGS_CPU) $(EXTRA_TESTFLAGS)

TESTFLAGS_TIMEOUT=10m
ifdef TIMEOUT
	TESTFLAGS_TIMEOUT=$(TIMEOUT)
endif

GOFILES = $(shell find . -name \*.go)

all: build

.PHONY: build
build:
	GO_BUILD_FLAGS="${GO_BUILD_FLAGS} -v -mod=readonly" ./scripts/build.sh

.PHONY: verify
verify: verify-gofmt verify-lint verify-mod-tidy

.PHONY: verify-gofmt
verify-gofmt:
	@echo "Verifying gofmt"
	@!(gofmt -l -s -d ${GOFILES} | grep '[a-z]')

	@echo "Verifying goimports"
	@!(go run golang.org/x/tools/cmd/goimports@latest -l -d ${GOFILES} | grep '[a-z]')

.PHONY: install-golangci-lint
install-golangci-lint:
	./scripts/verify_golangci-lint_version.sh

.PHONY: verify-lint
verify-lint: install-golangci-lint
	@echo "Verifying lint"
	golangci-lint run ./...

.PHONY: verify-mod-tidy
verify-mod-tidy:
	PASSES="mod_tidy" ./scripts/test.sh

.PHONY: test
test:
	CGO_ENABLED=${CGO_ENABLED} go test -v ${TESTFLAGS} -timeout ${TESTFLAGS_TIMEOUT} ./...

clean:
	rm -rf ./bin
	rm -rf ./release
	rm -rf ./.idea
	rm -f etcd-diagnosis
