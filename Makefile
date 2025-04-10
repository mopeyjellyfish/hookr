default: help

PROJECTNAME=$(shell basename "$(PWD)")

CLI_MAIN_FOLDER=./cmd/main.go
BIN_FOLDER=bin
BIN_FOLDER_MACOS=${BIN_FOLDER}/amd64/darwin
BIN_FOLDER_WINDOWS=${BIN_FOLDER}/amd64/windows
BIN_FOLDER_LINUX=${BIN_FOLDER}/amd64/linux
BIN_NAME=${PROJECTNAME}

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent
# LDFLAGS=-X main.buildDate=`date -u +%Y-%m-%dT%H:%M:%SZ` -X main.version=`scripts/version.sh`
LDFLAGS=-extldflags -static

## setup: install all build dependencies
setup: setup/tools download

## setup/tools: install all tools
setup/tools:
	@echo "  >  Installing tools"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v2.0.2
	go install github.com/kyoh86/richgo@latest

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
	@echo "  >  Linting..."
	@golangci-lint run

## test: run all unit tests
test:
	@echo "  >  Executing unit tests"
	@if ! type "richgo" > /dev/null 2>&1; then \
		go test -v -timeout 10s -race -coverprofile=coverage.txt -coverpkg=./... ./...; \
	else \
		richgo test -v -timeout 10s -race -coverprofile=coverage.txt -coverpkg=./... ./...; \
	fi

## test/cover: run all unit tests with coverage
test/cover: test
	go tool cover -html=./coverage.txt

## test/ff: run all tests fail on first failure
test/ff:
	@echo "  >  Executing unit tests - fail fast"
	@if ! type "richgo" > /dev/null 2>&1; then \
		go test -v -timeout 60s -race -failfast ./...; \
	else \
		richgo test -v -timeout 60s -race -failfast ./...; \
	fi

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo