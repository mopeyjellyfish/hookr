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

- an application-specific API like DICE's engine model,
- a reimplementation of FlatBuffers tooling,
- a string-dispatch plugin system,
- a framework that makes host/plugin authors manually wire transport details.

## Status Snapshot

Implemented in the repository now:

- [x] single installable `hookr` CLI with `gen`, `build`, `inspect`, and `bench`
- [x] FlatBuffers-first `hookr gen` path using `flatc`
- [x] `.bfbs`-driven internal contract loading and canonical hashing
- [x] Go FlatBuffers code generation with object API enabled
- [x] generated Go host SDK / plugin PDK glue for FlatBuffers contracts
- [x] TinyGo-first `hookr build`
- [x] working first-party fixtures:
  - `urlbalancer`
  - `textfilter`
  - `tickloop`
- [x] end-to-end tests covering:
  - plugin build
  - generated host open/call path
  - plugin -> host callbacks
  - optional plugin method behavior
- [x] first benchmark fixture on `tickloop`

Still future work:

- [ ] DICE pilot integration and migration
- [ ] explicit method inventory in handshake metadata
- [ ] richer benchmark matrix and published result snapshots
- [ ] schema + Wasm plugin inspection from the CLI
- [ ] ad hoc plugin invocation from the CLI
- [ ] generic host callback stubbing for development workflows
- [ ] fixture-driven plugin debugging workflows
- [ ] interactive TUI for plugin exploration
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
- DICE can later define its own contract and use the same system without
  Hookr learning DICE semantics.

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
rt, err := examplehookr.Open(ctx, examplehookr.Config{
	WasmPath: "./plugin.wasm",
	Host:     hostImpl,
})
if err != nil {
	return err
}

resp, err := rt.Balance(ctx, req)
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

- [x] `Plugin` and `Host` are the default service names
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
- `hookr build` should start TinyGo-first while leaving room for more build
  strategies later

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

rpc_service Host {
  RngInt(RngIntRequest):RngIntResponse;
}
```

The intended Hookr interpretation:

- `Plugin` methods are callable by the host
- `Host` methods are callbacks callable by the plugin
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

We should keep several example contracts in the repo so Hookr is proven against
more than one shape.

### Example 1: `urlbalancer`

Purpose:

- structured input/output
- plugin -> host callbacks
- realistic plugin logic with URL parsing and node selection

Contract shape:

- plugin methods: `GetInfo`, `Balance`
- host callbacks: `RngInt`, `RngFloat`

### Example 2: `textfilter`

Purpose:

- smallest readable quickstart contract
- pure host -> plugin calls with no host callbacks
- easiest docs/tutorial example

Possible contract shape:

- plugin methods: `GetInfo`, `Filter`
- host callbacks: none

### Example 3: `tickloop`

Purpose:

- tight-loop benchmark fixture
- exercise a consumer-defined hot path that looks closer to game/server
  update-style workloads
- benchmark request/response overhead under repeated calls

Possible contract shape:

- plugin methods: `GetInfo`, `Tick`
- host callbacks: optional small helper callbacks if needed

These examples serve three roles:

- executable documentation
- end-to-end test fixtures
- benchmark fixtures

## Proposed Repository Shape

This is the intended repo direction once the schema-driven path is underway.

Public entrypoints:

- [ ] `cmd/hookr` for the installable Cobra CLI
- [ ] exported runtime packages that generated code depends on
- [ ] exported Go PDK helpers that generated plugin code depends on

Internal implementation areas:

- [ ] `internal/cli/` for Cobra command wiring
- [ ] `internal/flatc/` for locating and invoking `flatc`
- [ ] `internal/contract/` for reading `.bfbs` and building Hookr's contract IR
- [ ] `internal/codegen/` for Hookr-specific generation
- [ ] `internal/codegen/go/` for Go backend templates and rendering
- [ ] `internal/abi/` for canonical ABI metadata and handshake definitions
- [ ] `internal/buildkit/` for optional build orchestration later
- [ ] `internal/bench/` for benchmark harness helpers later

Existing packages should be reshaped only when the schema-driven path requires it. We
should not churn package layout before the first end-to-end path is proven.

## Proposed Generated Output Shape

For a user contract such as `urlbalancer.fbs`, `hookr gen` should emit two
layers of output:

1. official FlatBuffers outputs produced by `flatc`
2. Hookr-generated glue produced by Hookr

Example output layout:

```text
gen/urlbalancer/
  flatbuffers/
    url_balancer_generated.go
  hookr/
    contract_meta_gen.go
    method_ids_gen.go
    host_sdk_gen.go
    plugin_pdk_gen.go
    runtime_gen.go
```

Design constraints:

- [ ] Hookr-generated files should clearly depend on official FlatBuffers files
- [ ] users should not edit any generated files
- [ ] generated package names should be predictable and documented
- [ ] we should avoid generating overlapping type names with `flatc`
- [ ] generation should be deterministic so CI diffs remain stable

## Contract Conventions

The contract model should be opinionated but generic.

Current intended convention:

- [x] one plugin-facing service, default name `Plugin`
- [x] zero or one host-callback service, default name `Host`
- [x] method request and response types must be explicit tables
- [x] method IDs are derived from canonical service + method identity
- [x] required/optional plugin methods are explicit metadata, not guesswork

We should decide early whether optionality comes from:

- [x] schema annotations
- [ ] generator config

We should keep optionality in the schema itself unless we hit a hard technical
constraint with FlatBuffers tooling.

Current intended choice:

- [x] declare a custom FlatBuffers attribute such as `hookr_optional`
- [x] attach it directly to `rpc_service` methods
- [ ] support non-schema optionality configuration in the first pass

## Implementation Phases

## Phase 0: Lock The Product Shape

Goal:

- freeze the Hookr product definition before writing more runtime code.

Deliverables:

- [ ] align `docs/plugin-system.md` and this plan
- [x] decide final CLI conventions
- [x] decide service naming conventions
- [x] decide required/optional method rules
- [ ] decide first benchmark matrix

Exit criteria:

- we can explain Hookr in one page without referencing internal draft APIs

## Phase 1: Toolchain Orchestration

Goal:

- make `hookr gen` depend on official FlatBuffers tooling instead of homegrown
  schema parsing.

Work:

- [ ] detect `flatc` from PATH or explicit flag
- [ ] add `hookr gen --schema ... --lang go`
- [ ] invoke `flatc --binary --schema` to emit `.bfbs`
- [ ] invoke `flatc` for Go bindings
- [ ] define output directory layout for generated FlatBuffers files and Hookr
      glue files
- [ ] produce actionable errors when `flatc` is missing
- [ ] add `hookr build` for TinyGo-first Go plugin builds
- [ ] keep the build command architecture open for future non-TinyGo build
      backends
- [ ] make build command output and errors suitable for docs/tutorial use

Exit criteria:

- `hookr gen` can produce `.bfbs` and Go FlatBuffers outputs for
  `urlbalancer.fbs`

## Phase 2: Contract IR And Canonical Hash

Goal:

- define Hookr's contract model based on `.bfbs`.

Work:

- [ ] read `.bfbs` into an internal contract model
- [ ] identify plugin service and host service
- [ ] normalize service/method/type metadata
- [ ] define method ID derivation rules
- [ ] define canonical contract hash from normalized contract metadata, not raw
      `.fbs` text
- [ ] define required vs optional method metadata from schema annotations

Exit criteria:

- formatting changes in `.fbs` do not change the contract hash
- semantic method or type changes do change the contract hash

Notes:

- the contract hash must not depend on comments, whitespace, field ordering
  noise from source text formatting, or unrelated schema file layout changes
- the contract hash must depend on the actual callable contract shape and any
  ABI-relevant annotations

## Phase 3: ABI Runtime Cleanup

Goal:

- make the runtime ABI fit the intended product shape.

Work:

- [ ] keep the hot path method-ID based
- [ ] remove reliance on string-dispatch in the normal generated path
- [ ] separate feature flags from method presence
- [ ] represent method presence in handshake metadata
- [ ] harden handshake parsing to avoid host panics
- [ ] ensure failed loads clean up runtime resources
- [ ] define arena ownership/reset rules clearly

Exit criteria:

- the runtime can validate a generated contract safely and predictably

Notes:

- feature flags should only represent transport/runtime features
- method presence should be reported as method inventory, not overloaded into
  feature bits
- host panics caused by malformed guest memory reads are release blockers

## Phase 4: Go Glue Generator

Goal:

- generate actual Go host/plugin glue for arbitrary consumer-defined
  contracts.

Work:

- [ ] generate contract metadata and method constants
- [ ] generate host callback interface
- [ ] generate plugin interface
- [ ] generate `Open(...)`
- [ ] generate `MustRegisterPlugin(...)`
- [ ] generate plugin context callback helpers
- [ ] generate request/response wrapper helpers over official FlatBuffers output
- [ ] preserve direct use of FlatBuffers-generated Go types in the public API
- [ ] add lower-level helper paths only where needed for hot-loop performance

Exit criteria:

- the `urlbalancer` contract can generate a complete Go host/plugin SDK/PDK

Notes:

- the first generated API should bias toward clear application code
- if needed, we can generate both:
  - ergonomic owned-object helpers
  - lower-level borrowed-view helpers for hot paths
- the normal tutorial path should stay small even if advanced performance APIs
  exist underneath

## Phase 5: UrlBalancer E2E Fixture

Goal:

- prove the system end to end with a consumer-defined example.

Scope:

- plugin method `GetInfo`
- plugin method `Balance`
- host callback `RngInt`
- host callback `RngFloat`

Work:

- [ ] keep `testdata/contracts/urlbalancer/urlbalancer.fbs` as the canonical
      fixture contract
- [ ] generate Go outputs into a fixture directory
- [ ] add example host implementation
- [ ] add example plugin implementation
- [ ] add e2e test that:
      - [ ] runs `hookr gen`
      - [ ] builds the example plugin
      - [ ] loads the plugin from a Go host
      - [ ] calls `GetInfo`
      - [ ] calls `Balance`
      - [ ] validates URL normalization and node selection
      - [ ] proves plugin -> host callback flow

Exit criteria:

- `urlbalancer` acts as both docs and a real e2e test fixture

Proposed fixture layout:

- [ ] `testdata/contracts/urlbalancer/urlbalancer.fbs`
- [ ] `testdata/contracts/urlbalancer/plugin/`
- [ ] `testdata/contracts/urlbalancer/host/`
- [ ] `testdata/contracts/urlbalancer/gen/`
- [ ] `testdata/contracts/urlbalancer/e2e_test.go`

Fixture expectations:

- [ ] host example should be readable as documentation
- [ ] plugin example should be readable as documentation
- [ ] e2e should exercise real generated code, not mocks
- [ ] fixture should prove both success and contract validation failure cases

Additional fixture work:

- [ ] add `testdata/contracts/textfilter/`
- [ ] add `testdata/contracts/tickloop/`
- [ ] ensure at least one example has no host callbacks
- [ ] ensure at least one example is tuned for hot-loop benchmarking

## Phase 6: Benchmarks

Goal:

- prove the new path with real measurements.

Benchmark categories:

- startup
  - [ ] compile
  - [ ] instantiate
  - [ ] handshake
- steady-state host -> plugin
  - [ ] small request/response
  - [ ] medium request/response
  - [ ] URL balancer request shape
- plugin -> host callbacks
  - [ ] `RngInt`
  - [ ] `RngFloat`
- loop scenarios
  - [ ] 10 Hz single runtime
  - [ ] 120 Hz single runtime
  - [ ] multiple runtime instances in parallel
- comparisons
  - [ ] previous runtime baseline snapshot
  - [ ] generated FlatBuffers path

Metrics:

- [ ] ns/op
- [ ] B/op
- [ ] allocs/op
- [ ] documented benchmark command lines
- [ ] benchmark result snapshots in `docs/benchmarks/`

Benchmark harness expectations:

- [ ] benchmark data should be reproducible from repo instructions
- [ ] each benchmark should identify payload size and call direction
- [ ] benchmark comparisons should state exact commands and machine details
- [ ] we should benchmark both happy-path invoke cost and host-callback cost

Exit criteria:

- we can defend the Hookr performance story with repeatable numbers

## Phase 7: DICE Pilot

Goal:

- validate Hookr against a real consumer with a non-trivial contract.

Work:

- [ ] design a DICE-specific `.fbs` contract
- [ ] map DICE plugin methods into that contract
- [ ] map DICE host callbacks into that contract
- [ ] generate DICE-specific Hookr glue
- [ ] integrate with DICE Wasm runtime and EDK
- [ ] benchmark DICE critical paths

Exit criteria:

- DICE can use Hookr without Hookr hardcoding DICE semantics

Notes:

- DICE should validate the genericity of the system, not define it
- DICE-specific helpers should live above Hookr-generated APIs, not inside
  Hookr runtime internals

## Phase 8: Multi-Language Foundations

Goal:

- make sure the Go-first design does not block Rust/Zig later.

Work:

- [ ] keep the ABI and handshake language-neutral
- [ ] isolate language-specific generation behind backends
- [ ] define what Hookr expects from generated FlatBuffers language bindings
- [ ] document how future backends fit into the pipeline

Exit criteria:

- adding Rust later is additive work, not a redesign

## Phase 9: Rust, Then Zig

Goal:

- add non-Go plugin-language support on top of the same contract/ABI.

Work:

- [ ] Rust backend design
- [ ] Rust PDK
- [ ] Rust e2e example
- [ ] Zig backend feasibility review
- [ ] Zig generation + PDK path once tooling choice is stable

Exit criteria:

- one `.fbs` contract can target multiple plugin languages

## Delivery Milestones

These are the practical milestones we should expect to land in order.

### Milestone 1: Toolchain proof

- [ ] `hookr gen` finds `flatc`
- [ ] `hookr gen` produces `.bfbs`
- [ ] `hookr gen` produces Go FlatBuffers bindings
- [ ] `hookr build` can compile a Go plugin example to Wasm

### Milestone 2: Contract proof

- [ ] Hookr reads `.bfbs`
- [ ] Hookr identifies plugin and host services
- [ ] Hookr computes stable method IDs and contract hash

### Milestone 3: Runtime proof

- [ ] generated Go host opens a generated plugin
- [ ] generated Go host calls a generated plugin method
- [ ] plugin calls a generated host callback

### Milestone 4: Documentation and e2e proof

- [ ] `urlbalancer` docs/example path is copy-pasteable
- [ ] `urlbalancer` e2e passes in CI
- [ ] failures are understandable when contracts do not match

### Milestone 5: Performance proof

- [ ] benchmark suite exists
- [ ] generated schema-driven path is measured against the current path
- [ ] results are documented before broader rollout

## Dependencies Between Phases

The phases are not independent. The critical path is:

1. Phase 1 must land before any real schema-driven generation.
2. Phase 2 must land before the generated handshake can be trusted.
3. Phase 3 and Phase 4 can proceed in parallel in parts, but both must land
   before the first credible e2e system exists.
4. Phase 5 should be the first place we judge the real user experience.
5. Phase 6 should run before we broaden the API or claim the performance story.
6. Phase 7 should happen only after the generic path is already coherent.

## Technical Decisions To Make

- [x] `Plugin` and `Host` are the permanent default service names
- [x] plugin methods are required by default unless explicitly marked optional
- [x] the Go API should prioritize ergonomic application code while preserving
      a performance path
- [x] `hookr build` is in scope for the first full CLI experience
- [x] generated Go APIs should delegate directly to FlatBuffers-generated types
- [x] we should add more than one first-party fixture before DICE
- [x] define the exact schema annotation form for optional methods
- [ ] define the exact Go package layout for generated FlatBuffers vs Hookr glue
- [x] define whether `hookr build` supports only TinyGo first, or multiple
      build strategies from the start

Chosen answers:

- [x] optional methods use a declared FlatBuffers custom attribute on the RPC
      method, for example `hookr_optional`
- [x] `hookr build` is TinyGo-first, with an internal design that can add more
      build strategies later

## Risks

- Over-coupling the public API to Go-specific types.
- Letting the generator depend on unstable assumptions about FlatBuffers output.
- Building too much convenience on top of generated types and losing the
  performance story.
- Designing around DICE too early and accidentally making Hookr app-specific.
- Promising Rust/Zig too early before the Go path is clean and benchmarked.

## What We Should Not Do

- Do not re-implement FlatBuffers parsing.
- Do not re-implement FlatBuffers wire serialization by hand.
- Do not make the public API depend on string method names.
- Do not make users manually register or hash methods.
- Do not let benchmarking wait until after feature completion.

## Recommended Next Execution Slice

The next slice should be deliberately narrow:

1. Make `hookr gen` invoke `flatc`.
2. Read `.bfbs`.
3. Build the internal contract model.
4. Generate Go glue for `urlbalancer`.
5. Add one real e2e host/plugin test.

This is the smallest slice that starts proving the actual system instead of
just planning it.

## Remaining Clarifications

These are the next questions that still affect implementation detail.

1. What exact generated Go package layout do you want for official
   FlatBuffers-generated files versus Hookr glue files?
2. Do you want `hookr build` to expose the build backend explicitly from day
   one, or should the first version keep the CLI simpler and default to TinyGo?
3. Do you want Hookr to support service-name overrides in the first pass, or
   should we keep the first implementation strict around `Plugin` and `Host`
   until the main path is proven?
