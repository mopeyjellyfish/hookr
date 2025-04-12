
# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


## build: build the WASM binary
.PHONY: build
build:
	tinygo build -o bin/simple.wasm -scheduler=none -target=wasi -buildmode=wasi-legacy simple.go
