# Hookr Implementation Plan

This document belongs to the Diataxis Explanation section:
[Explanation Index](./explanation/index.md).

This document is the working plan for building Hookr as a generic plugin
library for Go applications.

It is intended to be the execution document for the work after the initial
direction-setting phase.

## Summary

Hookr should be:

- a generic Wasm plugin framework,
- driven by user-defined FlatBuffers contracts,
- built around generated host SDK and plugin PDK glue,
- optimized for tight-loop calls,
- designed for Go first and future multi-language PDKs later.

Hookr should not be:

- an application-specific engine or workflow model,
- a reimplementation of FlatBuffers tooling,
- a string-dispatch plugin system,
- a framework that makes host/plugin authors manually wire transport details.

## Status Snapshot

Implemented in the repository now:

- [x] single installable `hookr` CLI with `gen`, `build`, `inspect`, `call`,
      `bench`, and `tui`
- [x] FlatBuffers-first `hookr gen` path using `flatc`
- [x] `.bfbs`-driven internal contract loading and canonical hashing
- [x] Go FlatBuffers code generation with object API enabled
- [x] generated Go host SDK / plugin PDK glue for FlatBuffers contracts
- [x] auto-discovered host callback modules from every non-`Plugin`
      `rpc_service`
- [x] standard-Go `hookr build`
- [x] working first-party fixtures:
  - `urlbalancer`
  - `textfilter`
  - `tickloop`
- [x] end-to-end tests covering:
  - plugin build
  - generated host open/call path
  - plugin -> host callbacks
  - optional plugin method behavior
- [x] schema + Wasm plugin inspection from the CLI
- [x] ad hoc plugin invocation from the CLI
- [x] generic host callback stubbing for development workflows
- [x] fixture-driven plugin debugging workflows
- [x] interactive TUI for plugin exploration
- [x] initial benchmark fixtures and published benchmark snapshots

Still future work:

- [ ] reference consumer integration and migration
- [ ] explicit method inventory in handshake metadata
- [ ] richer benchmark matrix and published result snapshots
- [ ] Rust backend / PDK
- [ ] Zig backend / PDK

## Context

Current repository state:

- The repo already has a method-ID ABI draft and generator skeleton.
- The repo now has a single installable Cobra CLI entrypoint at `cmd/hookr`.
- There is now an initial direction document in `docs/plugin-system.md`.
- There is now an initial consumer-defined fixture contract in
  `testdata/contracts/urlbalancer/urlbalancer.fbs`.

Key correction made during planning:

- `Definition`, `Update`, and similar methods are not Hookr concepts.
- They are methods defined by an application that uses Hookr.
- Hookr's job is to transport, validate, and generate glue around those methods.

## Product Definition

Hookr owns:

- the Wasm ABI,
- runtime and PDK shims,
- startup handshake and contract validation,
- code generation orchestration,
- generated host SDK and plugin PDK glue,
- benchmarking and e2e fixtures.

Hookr does not own:

- application-specific plugin methods,
- FlatBuffers schema parsing,
- FlatBuffers wire-format implementation,
- language-native FlatBuffers bindings.

## Non-Negotiable Decisions

- [x] FlatBuffers is the single contract and wire format.
- [x] Hookr uses `flatc` and existing FlatBuffers libraries instead of
      implementing FlatBuffers itself.
- [x] Hookr uses `.bfbs` as the internal contract IR for generation.
- [x] Method-ID dispatch is the only hot-path dispatch model.
- [x] Generated code is the default public API.
- [x] Go is the first supported language for host + plugin.
- [x] Rust and Zig are planned, but not allowed to distort the Go-first API.

## Success Criteria

Hookr is successful when:

- a host project writes one `.fbs` contract and little else,
- a plugin author implements generated interfaces and calls one register
  function,
- a host author implements generated host callbacks and calls one open
  function,
- the hot path benchmarks stay comfortably within the current runtime targets,
- the `urlbalancer` example proves the generic system end to end,
- a consuming application can later define its own contract and use the same
  system without Hookr learning application-specific semantics.

## Constraints

- The public path should be easy to explain in a short tutorial.
- The runtime must remain usable for tight-loop workloads.
- We should minimize long-term maintenance cost.
- We should avoid painting ourselves into a Go-only design corner.
- We should not depend on handwritten method registration or schema-specific
  manual glue.

## Public UX Target

The intended developer flow:

1. Define a plugin system contract in `.fbs`.
2. Run `hookr gen`.
3. Implement the generated plugin interface.
4. Implement the generated host callback interface.
5. Build the plugin to Wasm.
6. Open the plugin using the generated host SDK.
7. Call typed methods.

The intended CLI surface:

- `hookr gen`
- `hookr build`
- `hookr inspect`
- `hookr call`
- `hookr bench`
- `hookr tui`

The developer-tooling expansion plan for `inspect`, `call`, callback stubbing,
fixtures, and the TUI is documented in
[Plugin Debugging And Development Tooling](./explanation/plugin-debugging-tooling.md).

The intended host-side Go feel:

```go
plugin, err := examplehookr.Open(ctx, examplehookr.Config{
	PluginPath: "./plugin.wasm",
	Host: examplehookr.Host{
		Rng: host{},
	},
})
if err != nil {
	return err
}

resp, err := plugin.Balance(ctx, req)
```

The intended plugin-side Go feel:

```go
type Plugin struct{}

func (Plugin) Balance(
	ctx *examplehookr.PluginContext,
	req examplehookr.BalanceRequest,
) (examplehookr.BalanceResponse, error) {
	// plugin logic
}

//go:wasmexport hookr_init
func hookrInit() {
	examplehookr.MustRegisterPlugin(Plugin{})
}
```

The user should not manually:

- assign method IDs,
- register methods by string,
- compute schema hashes,
- write serialization code,
- manage transport buffers,
- wire startup validation.

## Decisions Locked In

These decisions are now treated as defaults for implementation unless they
prove unworkable in the first end-to-end slice.

- [x] `Plugin` is the default plugin service name
- [x] every non-`Plugin` `rpc_service` is auto-discovered as a host module
- [x] plugin methods are required by default
- [x] optionality should be defined in the schema
- [x] the first Go API should be ergonomic while preserving a clear
      performance path
- [x] `hookr build` is part of the first full CLI story
- [x] generated Go APIs should use the FlatBuffers-generated types directly
- [x] we should maintain multiple example contracts, not only `urlbalancer`

Practical meaning:

- the normal application-facing API should remain small and readable
- the generated code should expose official FlatBuffers-generated request and
  response types in Go
- performance-oriented helpers can exist, but they should not make the primary
  tutorial path feel low-level
- the CLI should carry the full developer path from generation to plugin build
- plugin optionality should use FlatBuffers-compatible custom attributes on RPC
  methods
- `hookr build` should own the Go `wasip1/wasm` plugin path directly

## FlatBuffers Contract Example

This example shows the intended consumer-defined schema shape. The contract is
owned by the application using Hookr, not by Hookr itself.

```fbs
namespace Example.UrlBalancer;

table Empty {}

table PluginInfo {
  name:string;
  version:string;
  description:string;
}

table BalanceRequest {
  url:string;
  nodes:[string];
}

table BalanceResponse {
  valid:bool;
  normalized_url:string;
  selected_node:string;
  selected_index:uint32;
  error:string;
}

table RngIntRequest {
  min:int32;
  max:int32;
}

table RngIntResponse {
  value:int32;
}

rpc_service Plugin {
  GetInfo(Empty):PluginInfo;
  Balance(BalanceRequest):BalanceResponse;
}

rpc_service Rng {
  Int(RngIntRequest):RngIntResponse;
}
```

The intended Hookr interpretation:

- `Plugin` methods are callable by the host
- every non-`Plugin` service is a host callback module callable by the plugin
- Hookr generates method IDs, contract metadata, runtime glue, and typed Go
  APIs on top of the official FlatBuffers Go outputs

Optional methods should use declared FlatBuffers attributes, for example:

```fbs
attribute "hookr_optional";

rpc_service Plugin {
  GetInfo(Empty):PluginInfo;
  Warmup(Empty):Empty (hookr_optional);
}
```

## Example Contract Matrix

Hookr now uses three first-party contracts to prove the generic model:

### `urlbalancer`

Purpose:

- structured input/output
- plugin -> host callbacks
- a readable tutorial and debugger fixture

Current shape:

- plugin methods: `GetInfo`, `Balance`
- host module: `Rng`
- callback methods: `Int`, `Float`

### `textfilter`

Purpose:

- smallest readable quickstart contract
- pure host -> plugin calls with no host callbacks

Current shape:

- plugin methods: `GetInfo`, `Filter`
- host modules: none

### `tickloop`

Purpose:

- hot-loop benchmark fixture
- optional plugin method behavior
- low-allocation generated fast path

Current shape:

- plugin methods: `GetInfo`, `Tick`, optional `Warmup`
- host module: `Rng`
- callback method: `Int`

These examples serve three roles:

- executable documentation
- end-to-end test fixtures
- benchmark fixtures

## Repository Shape

The repo direction described earlier is largely in place now.

Public entrypoints:

- [x] `cmd/hookr` for the installable Cobra CLI
- [x] exported runtime packages that generated code depends on
- [x] exported Go PDK helpers that generated plugin code depends on

Internal implementation areas:

- [x] `internal/cli/` for Cobra command wiring
- [x] `internal/flatc/` for locating and invoking `flatc`
- [x] `internal/contract/` for reading `.bfbs` and building Hookr's contract IR
- [x] `internal/codegen/` for Hookr-specific generation
- [x] `internal/buildkit/` for Go `wasip1/wasm` build orchestration
- [x] `internal/bench/` for benchmark helpers
- [x] `internal/call/`, `internal/devhost/`, and `internal/tui/` for developer
      workflows

What remains is not package-shape work. It is product-depth work on a real
consumer integration,
benchmarks, and future language backends.

## Contract Conventions

The current convention is locked in:

- [x] one plugin-facing service, default name `Plugin`
- [x] every non-`Plugin` `rpc_service` is auto-discovered as a host module
- [x] method request and response types must be explicit tables
- [x] method IDs are derived from canonical service + method identity
- [x] required/optional plugin methods are explicit metadata
- [x] optionality is defined in the schema, not generator config
- [x] optional methods use a FlatBuffers custom attribute such as
      `hookr_optional`

## Current Phase Status

The original phased roadmap is now mostly complete through the Go-first
delivery track.

### Completed

- [x] Product shape locked around FlatBuffers, generated APIs, and one Hookr CLI
- [x] Toolchain orchestration through `flatc`
- [x] Contract IR and canonical hashing
- [x] ABI/runtime cleanup for the generated path
- [x] Go glue generator
- [x] end-to-end example fixtures
- [x] core developer tooling: inspect, call, host fixtures, TUI
- [x] initial benchmarking and benchmark documentation

### Remaining

- [ ] explicit method inventory in handshake metadata as a first-class
      documented feature
- [ ] richer benchmark matrix and published snapshots for more payload shapes
      and parallel scenarios
- [ ] reference consumer contract and migration
- [ ] multi-language backend design work

## Remaining Work

## Phase A: Benchmark Expansion

Goal:

- make the performance story broader and more repeatable than the current
  fixture benchmarks.

Work:

- [ ] add more startup benchmarks for compile, instantiate, and handshake
- [ ] add wider request/response size benchmarks
- [ ] add more plugin -> host callback benchmark cases
- [ ] add multi-runtime and multi-core comparison scenarios
- [ ] publish exact commands and machine details alongside snapshots

Exit criteria:

- benchmark docs clearly show steady-state invoke cost, callback cost, and
  scaling characteristics

## Phase B: Reference Consumer Pilot

Goal:

- validate Hookr as the runtime substrate for a real consuming application
  without adding application semantics to Hookr itself.

Work:

- [ ] design a consumer-specific `.fbs` contract
- [ ] model plugin lifecycle methods in that contract
- [ ] model consumer host callback modules in that contract
- [ ] decide which payloads stay opaque versus strongly modeled FlatBuffers
- [ ] generate consumer-specific Hookr glue
- [ ] integrate one real runtime path against generated Hookr APIs
- [ ] measure consumer-relevant critical paths

Exit criteria:

- a real consuming application can use Hookr through generated contracts and a
  thin adapter layer

## Phase C: Multi-Language Foundations

Goal:

- keep the current Go-first success from turning into a Go-only design.

Work:

- [ ] document backend boundaries for future code generation
- [ ] define what Hookr expects from generated non-Go PDK/runtime shims
- [ ] keep ABI and handshake details language-neutral in docs and code layout
- [ ] prepare one Rust backend design document

Exit criteria:

- adding Rust later is additive work, not a redesign

## Delivery Milestones From Here

### Milestone 1: Benchmark proof

- [ ] expanded benchmark suite exists
- [ ] published result snapshots cover hot loop, callbacks, and startup
- [ ] benchmark commands are copy-pasteable from the docs

### Milestone 2: Consumer proof

- [ ] consumer contract drafted
- [ ] one real runtime path implemented on Hookr
- [ ] consumer-relevant benchmark numbers captured

### Milestone 3: Backend proof

- [ ] future language backend boundary documented
- [ ] first Rust design slice drafted

## Risks

- Over-coupling the public API to Go-specific types.
- Letting the generator depend on unstable assumptions about FlatBuffers output.
- Building too much convenience on top of generated types and losing the
  performance story.
- Designing around one consumer too early and accidentally making Hookr app-specific.
- Promising Rust/Zig too early before the Go path is fully benchmarked.

## What We Should Not Do

- Do not re-implement FlatBuffers parsing.
- Do not re-implement FlatBuffers wire serialization by hand.
- Do not make the public API depend on string method names.
- Do not make users manually register or hash methods.
- Do not let benchmarking wait until after feature completion.
- Do not let consumer-specific helper APIs leak into Hookr core.

## Recommended Next Execution Slice

The next useful slice is:

1. cleanly define the consumer contract shape
2. expand the benchmark matrix around current fixtures
3. keep the docs and benchmark snapshots in sync with measured reality

That is the narrowest path that proves Hookr against a real consumer without
reopening foundational design work.

## Remaining Clarifications

These are the main questions that still affect the next phase of work.

1. How much plugin state should stay opaque bytes versus explicit
   FlatBuffers tables?
2. Which lifecycle methods should be required versus optional in the
   first contract?
3. Do we want Rust as the first non-Go plugin backend, or should backend work
   wait until after the reference consumer pilot?
