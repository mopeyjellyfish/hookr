# Agent Index

This page is the canonical entrypoint for LLMs, code assistants, retrieval
systems, and search tooling that need a compact map of Hookr.

If you are indexing the published docs site, also fetch:

- `/llms.txt`: short machine-oriented overview and canonical links
- `/llms-full.txt`: dense single-file project summary
- `/agent-index.json`: structured task and source map

## Project Summary

Hookr is a schema-defined WebAssembly plugin system for Go. A host application
defines a FlatBuffers contract, runs `hookr gen` to generate a typed Go host
SDK and plugin PDK, builds plugins to Wasm, and loads them through the Hookr
runtime. Hookr owns the Wasm ABI, compatibility handshake, method-ID dispatch,
trust policy, generated glue, and developer tooling.

Core model:

- one configured plugin `rpc_service`
- every other `rpc_service` is a host callback module
- FlatBuffers defines requests, responses, and method metadata
- generated Go code provides typed host and plugin APIs
- the runtime validates ABI version, schema hash, capabilities, and methods

## Canonical Reading Order

Use this order when building context from scratch:

1. [Overview](/)
2. [Contract Model](/reference/contracts)
3. [Generated Go API](/reference/generated-go-api)
4. [CLI Reference](/reference/cli)
5. [Architecture](/explanation/architecture)
6. [Live Reload Lifecycle](/explanation/live-reload-lifecycle)
7. [Plugin Debugging And Development Tooling](/explanation/plugin-debugging-tooling)

## Task Map

### Install and confirm the CLI

- [README](https://github.com/mopeyjellyfish/hookr/blob/main/README.md)
- [CLI Reference](/reference/cli)
- command: `go install github.com/mopeyjellyfish/hookr/cmd/hookr@latest`
- command: `hookr version`

### Learn the contract model

- [Contract Model](/reference/contracts)
- [Architecture](/explanation/architecture)
- [ABI Reference](/reference/abi)

### Generate code from a schema

- [Generate Glue](/how-to/generate-glue)
- [Generated Go API](/reference/generated-go-api)
- command: `hookr gen --schema ./contract.fbs --out ./gen --package myhookr`

### Build a plugin

- [Build Plugin](/how-to/build-plugin)
- command: `hookr build --plugin ./plugin --out ./bin/plugin.wasm`

### Open and call a plugin from Go

- [Open And Call Plugin](/how-to/open-and-call-plugin)
- [Generated Go API](/reference/generated-go-api)

### Implement host callbacks

- [Generated Go API](/reference/generated-go-api)
- [Open And Call Plugin](/how-to/open-and-call-plugin)
- Host callbacks are every `rpc_service` other than the configured plugin
  service.

### Inspect or debug a plugin artifact

- [Inspect Contract](/how-to/inspect-contract)
- [Debug Plugin From CLI](/how-to/debug-plugin-from-cli)
- [CLI Reference](/reference/cli)

### Enable live reload

- [Enable Live Reload](/how-to/enable-live-reload)
- [Live Reload Lifecycle](/explanation/live-reload-lifecycle)

### Understand performance expectations

- [Performance Model](/explanation/performance-model)
- [Benchmark Reference](/reference/benchmarks)
- [Run Benchmarks](/how-to/run-benchmarks)

### Understand the release process

- [Roadmap And Release](/explanation/roadmap-and-release)
- [docs workflow](https://github.com/mopeyjellyfish/hookr/blob/main/.github/workflows/docs.yml)
- [release workflow](https://github.com/mopeyjellyfish/hookr/blob/main/.github/workflows/release.yml)
- [semantic-release config](https://github.com/mopeyjellyfish/hookr/blob/main/.releaserc.json)

## Canonical Source Files

When retrieval needs code, these are the highest-signal files:

- [`internal/cli/root.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/cli/root.go)
- [`runtime/runtime.go`](https://github.com/mopeyjellyfish/hookr/blob/main/runtime/runtime.go)
- [`runtime/live.go`](https://github.com/mopeyjellyfish/hookr/blob/main/runtime/live.go)
- [`runtime/file.go`](https://github.com/mopeyjellyfish/hookr/blob/main/runtime/file.go)
- [`internal/contract/contract.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/contract/contract.go)
- [`internal/contract/canonical_hash.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/contract/canonical_hash.go)
- [`internal/codegen/flatbuffers.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/codegen/flatbuffers.go)
- [`internal/buildkit/build.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/buildkit/build.go)
- [`internal/inspect/inspect.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/inspect/inspect.go)
- [`internal/call/call.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/call/call.go)
- [`internal/tui/tui.go`](https://github.com/mopeyjellyfish/hookr/blob/main/internal/tui/tui.go)
- [`pdk/pdk.go`](https://github.com/mopeyjellyfish/hookr/blob/main/pdk/pdk.go)

## First-Party Example Contracts

Use these when you need working examples instead of abstract reference docs:

- [textfilter](https://github.com/mopeyjellyfish/hookr/blob/main/testdata/contracts/textfilter/README.md):
  minimal plugin without host callbacks
- [urlbalancer](https://github.com/mopeyjellyfish/hookr/blob/main/testdata/contracts/urlbalancer/README.md):
  host callback modules, inspect/call/TUI fixtures
- [tickloop](https://github.com/mopeyjellyfish/hookr/blob/main/testdata/contracts/tickloop/README.md):
  hot-loop and benchmark-oriented contract

## Retrieval Notes

- Prefer the reference docs for exact behavior and flags.
- Prefer the explanation docs for design rationale and tradeoffs.
- Prefer `testdata/contracts/*` when you need end-to-end examples.
- Prefer `runtime/*` and `internal/codegen/*` when you need implementation
  details.
- Treat older top-level design notes like
  [plugin-system.md](https://github.com/mopeyjellyfish/hookr/blob/main/docs/plugin-system.md) and
  [implementation-plan.md](https://github.com/mopeyjellyfish/hookr/blob/main/docs/implementation-plan.md)
  as supporting background, not first-pass entrypoints.
