default: help

PROJECTNAME=$(shell basename "$(PWD)")

CLI_MAIN_FOLDER=./cmd/hookr
BIN_FOLDER=bin
BIN_FOLDER_MACOS=${BIN_FOLDER}/amd64/darwin
BIN_FOLDER_WINDOWS=${BIN_FOLDER}/amd64/windows
BIN_FOLDER_LINUX=${BIN_FOLDER}/amd64/linux
BIN_NAME=${PROJECTNAME}
GOTESTSUM ?= gotestsum
TEST_PKGS=./runtime/...
TEST_COVERFLAGS=-timeout 30s -race -coverprofile=coverage.txt -coverpkg=./runtime/...
TEST_FAILFAST_FLAGS=-timeout 60s -race -failfast

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent
BASE_LDFLAGS=-extldflags -static
LDFLAGS=$(BASE_LDFLAGS)
ifneq ($(strip $(VERSION)),)
LDFLAGS=$(BASE_LDFLAGS) -X github.com/mopeyjellyfish/hookr/internal/version.Value=$(VERSION)
endif

## setup: install all build dependencies
setup: setup/go setup/tools download

## setup/tools: install all tools
setup/tools:
	@echo "  >  Installing tools"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v2.0.2

# setup/go: install go tooling
setup/go:
	@echo "  >  Installing go tools"
	go install gotest.tools/gotestsum@latest

## compile: compiles project in current system
compile: clean fmt vet lint test build

## clean: remove all build artifacts
clean:
	@echo "  >  Cleaning build cache"
	@-rm -rf ${BIN_FOLDER}/amd64 ${BIN_FOLDER}/${BIN_NAME} \
		&& go clean ./...

## build: build the binary
build:
	@echo "  >  Building binary"
	@go build \
		-ldflags="${LDFLAGS}" \
		-o ${BIN_FOLDER}/${BIN_NAME} \
		"${CLI_MAIN_FOLDER}"

## docs/install: install documentation site dependencies
docs/install:
	@echo "  >  Installing docs dependencies"
	@npm install

## docs/serve: run the local VitePress docs server
docs/serve: docs/install
	@echo "  >  Starting docs server"
	@npm run docs:dev

## docs/build: build the VitePress docs site
docs/build: docs/install
	@echo "  >  Building docs site"
	@npm run docs:build

## docs/preview: preview the built VitePress docs site
docs/preview: docs/build
	@echo "  >  Previewing docs site"
	@npm run docs:preview

## hooks/install: install repo-local git hooks
hooks/install:
	@echo "  >  Installing git hooks"
	@chmod +x .githooks/pre-commit
	@git config core.hooksPath .githooks

## hooks/run: run the pre-commit checks
hooks/run:
	@echo "  >  Running pre-commit checks"
	@.githooks/pre-commit

## build/all: build the binary for all platforms
build/all: build/macos build/windows build/linux

## build/macos: build the binary for MacOS
build/macos:
	@echo "  >  Building binary for MacOS"
	@GOOS=darwin GOARCH=amd64 \
		go build \
		-ldflags="${LDFLAGS}" \
		-o ${BIN_FOLDER_MACOS}/${BIN_NAME} \
		"${CLI_MAIN_FOLDER}"

## build/windows: build the binary for Windows
build/windows:
	@echo "  >  Building binary for Windows"
	@GOOS=windows GOARCH=amd64 \
		go build \
		-ldflags="${LDFLAGS}" \
		-o ${BIN_FOLDER_WINDOWS}/${BIN_NAME}.exe \
		"${CLI_MAIN_FOLDER}"

## build/linux: build the binary for Linux
build/linux:
	@echo "  >  Building binary for Linux"
	@GOOS=linux GOARCH=amd64 \
		go build \
		-ldflags="${LDFLAGS}" \
		-o ${BIN_FOLDER_LINUX}/${BIN_NAME} \
		"${CLI_MAIN_FOLDER}"



## tidy: clean up go.mod and go.sum files
tidy:
	@echo "  >  Tidy & Verify go.mod and go.sum files"
	@go mod tidy
	@go mod verify

## download: download all dependencies
download:
	@echo "  >  Download dependencies..."
	@go mod download && go mod tidy

## fmt: format all go files
fmt:
	@echo "  >  Formatting..."
	@go fmt ./...

## vet: run go vet
vet:
	@echo "  >  Vet..."
	@go vet ./...

## lint: run golangci-lint
lint:
	@echo "  >  Linting ./runtime..."
	@golangci-lint run ./runtime/...

## test: run all unit tests
test:
	@echo "  >  Executing unit tests"
	@$(GOTESTSUM) --format pkgname -- $(TEST_COVERFLAGS) $(TEST_PKGS)

## test/cover: run all unit tests with coverage
test/cover: build/testdata test
	go tool cover -html=./coverage.txt

## test/ff: run all tests fail on first failure
test/ff:
	@echo "  >  Executing unit tests - fail fast"
	@$(GOTESTSUM) --format pkgname -- $(TEST_FAILFAST_FLAGS) $(TEST_PKGS)

## build/runtime: build the runtime for hookr to be injected into the WASM runtime
build/runtime:
	@echo "  >  Building hookr WASM runtime"
	@tinygo build -o pkg/host/runtime.wasm -scheduler=none -target=wasip1 runtime/main.go

## build/testdata: build all test data WASM modules
build/testdata:
	@echo "  >  Building all test data WASM modules"
	@for dir in $(shell find ./testdata -type d -mindepth 1 -maxdepth 1); do \
		if [ -f $$dir/Makefile ]; then \
			echo "  >  Building $$(basename $$dir) test data"; \
			(cd $$dir && make build); \
		fi \
	done

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
