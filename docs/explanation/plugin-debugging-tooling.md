# Plugin Debugging And Development Tooling

Hookr can already generate bindings, build plugins, inspect schema-derived
contract metadata, and run benchmarks. The next missing loop is day-to-day
plugin development tooling: loading a plugin module, validating it against a
contract, making ad hoc calls, stubbing host callbacks, and exploring behavior
without writing a full host application first.

This document defines that plan.

## Problem

Today Hookr is strong at:

- generating Go host SDK and plugin PDK glue from FlatBuffers
- building TinyGo-based plugin Wasm binaries
- opening plugins from real host code
- benchmarking fixture contracts

Today Hookr is weak at:

- inspecting a `.wasm` plugin against a schema from the CLI
- listing implemented methods from the actual loaded module
- making ad hoc method calls into a plugin
- supplying temporary host callback behavior during development
- iterating on request and response payloads without a custom host
- debugging optional-method behavior and handshake state

That gap matters because Hookr is intended to be the foundation for other
applications. A host application should define the contract, but Hookr should
still give plugin and host developers a strong workflow for:

- validating a plugin quickly
- reproducing calls outside the real host
- exploring contract changes safely
- debugging callback-heavy behavior
- sharing request fixtures and test cases

## Goals

The tooling should let a developer:

1. validate a plugin Wasm binary against a FlatBuffers contract
2. inspect ABI version, schema hash, capabilities, and implemented methods from
   the actual plugin
3. call plugin methods from the command line using human-friendly input
4. supply host callback behavior without writing a full host application
5. save and replay request and callback fixtures
6. graduate from non-interactive CLI flows to an interactive TUI

## Non-Goals

This tooling should not:

- replace the generated Go SDK as the main production integration path
- expose raw FlatBuffers binary payloads as the human-facing interface
- become a second contract system alongside `.fbs`
- add application semantics to Hookr itself
- require handwritten transport or registration code

## Core Design Principles

### FlatBuffers for transport, JSON/forms for humans

Runtime transport should stay FlatBuffers. Human interaction should not.

The CLI and TUI should accept JSON-like input and present JSON-like output,
then use the schema reflection data to map that data to FlatBuffers behind the
scenes.

### Schema plus plugin is the unit of inspection

Inspecting the schema alone is useful. Inspecting a plugin module alone is not
enough. The main developer loop should use both:

- `contract.fbs`
- `plugin.wasm`

That lets Hookr answer the questions developers actually care about:

- does this module match the schema?
- which plugin methods are implemented?
- which methods are optional and missing?
- what capabilities and handshake values does the plugin publish?

### CLI first, TUI second

The CLI gives the fastest path to correctness and automation:

- easy to script in CI
- easy to reproduce with saved fixtures
- easiest way to define stable request/response semantics

The TUI should be built only after the CLI path is solid.

### Generic host stubbing, not app-specific behavior

Hookr must stay generic. Any callback stubbing system should be contract-driven
and schema-aware, not hand-coded for specific applications.

## Target User Flows

### Flow 1: Validate a plugin

The developer wants to confirm that a plugin module matches the expected
contract before wiring it into a host.

Target command:

```bash
hookr inspect \
  --schema ./contract.fbs \
  --wasm ./plugin.wasm
```

Expected output:

- schema hash from the contract
- schema hash published by the plugin
- ABI version
- capabilities
- implemented plugin methods
- required plugin methods
- optional plugin methods
- any mismatches or missing required methods

### Flow 2: Call a plugin method directly

The developer wants to invoke a plugin method with a saved request fixture.

Target command:

```bash
hookr call \
  --schema ./contract.fbs \
  --wasm ./plugin.wasm \
  --method Balance \
  --input ./requests/balance.json
```

Expected behavior:

- load the plugin
- validate the contract and handshake
- convert JSON to the method's request FlatBuffer
- invoke the plugin method
- decode the response
- print JSON

### Flow 3: Call a plugin with host callback stubs

The developer wants to test a plugin without writing a custom host.

Target command:

```bash
hookr call \
  --schema ./contract.fbs \
  --wasm ./plugin.wasm \
  --method Balance \
  --input ./requests/balance.json \
  --host-fixture ./fixtures/host.json
```

Expected behavior:

- Hookr loads callback definitions from a fixture file
- callback responses can be static, deterministic, or sequential
- plugin methods can call the `Host` service through a generated or reflection
  driven development host

### Flow 4: Explore a plugin interactively

The developer wants to browse methods, tweak requests, and re-run calls quickly.

Target command:

```bash
hookr tui \
  --schema ./contract.fbs \
  --wasm ./plugin.wasm
```

Expected behavior:

- browse plugin methods
- inspect request and response fields
- edit request JSON
- invoke methods repeatedly
- see callback activity and validation errors
- save fixtures from the session

## Phased Plan

### Phase 1: Deep Inspect

Add a Wasm-aware inspect path.

Deliverables:

- `hookr inspect --schema ... --wasm ...`
- schema-only inspect remains supported
- output includes handshake metadata and implemented methods
- clear mismatch reporting for:
  - ABI version
  - schema hash
  - missing required methods
  - unknown published methods

Implementation notes:

- extend the existing `internal/inspect` package instead of creating a second
  inspection flow
- reuse the existing runtime loader and contract metadata parsing
- support both human-readable text and machine-readable JSON output

Acceptance criteria:

- a developer can validate a plugin module without writing host code
- CI can use `hookr inspect` as a contract gate

### Phase 2: Ad Hoc Invocation

Add a non-interactive `hookr call` command.

Deliverables:

- `hookr call --schema --wasm --method --input`
- request JSON loaded from file or stdin
- response JSON written to stdout
- support for methods on the default `Plugin` service

Implementation notes:

- create a new `internal/call` package
- use `.bfbs` reflection data to map JSON to request builders
- use the same reflection data to map response buffers back to JSON
- keep the first pass focused on tables, scalars, strings, and vectors

Acceptance criteria:

- every first-party example contract can be called via `hookr call`
- request and response fixtures can be stored in the repo

### Phase 3: Development Host Stubs

Add generic host callback stubbing.

Deliverables:

- `--host-fixture` support for `hookr call`
- static response stubs
- sequential response stubs
- deterministic random support for callback-heavy contracts

Implementation notes:

- add `internal/devhost` or `internal/stubhost`
- the fixture format should be contract-driven and method-oriented
- callback invocation logs should be capturable for debugging

Fixture shape example:

```json
{
  "RngInt": [
    { "value": { "value": 1 } },
    { "value": { "value": 3 } }
  ],
  "RngFloat": {
    "mode": "constant",
    "value": { "value": 0.5 }
  }
}
```

Acceptance criteria:

- callback-heavy fixtures like `urlbalancer` can be exercised from the CLI
- callback behavior is deterministic when driven by fixtures

### Phase 4: Fixture Workflow

Make requests and callback behavior reusable.

Deliverables:

- request fixture conventions in the examples
- response snapshot output option
- callback trace output option
- fixture directories that can be checked into source control

Implementation notes:

- standardize file layout for examples and docs
- make fixtures easy to consume in tests and docs

Acceptance criteria:

- docs examples can be run directly from checked-in fixtures
- developers can reproduce bugs using fixture files alone

### Phase 5: Interactive TUI

Add an optional interactive UI on top of the validated CLI flow.

Deliverables:

- `hookr tui`
- method list
- request editor
- invoke panel
- response panel
- callback trace panel

Implementation notes:

- build on the exact same invocation and fixture engine used by `hookr call`
- do not build a second execution stack just for the TUI
- start simple and schema-aware rather than attempting a full IDE

Acceptance criteria:

- a developer can load a plugin and inspect behavior without leaving Hookr
- the TUI uses the same schema and fixtures as the CLI

## Architecture Changes Needed

### Reuse and extend existing packages

The current repo already has useful building blocks:

- `internal/contract`
- `internal/inspect`
- `internal/flatc`
- `runtime`

The plan should extend those pieces rather than introduce parallel paths.

### New packages likely needed

- `internal/call`
  - coordinates schema loading, plugin loading, invocation, and output
- `internal/jsonfb`
  - reflection-driven JSON to FlatBuffers and FlatBuffers to JSON helpers
- `internal/devhost`
  - generic callback stubbing and fixture playback

The exact names can change, but the separation should be maintained.

### Reflection-driven bridge

The core enabling layer is a reflection-driven bridge between:

- JSON-friendly request/response values
- FlatBuffers request/response payloads

That bridge is the main technical requirement for both `hookr call` and the
future TUI.

## Command Surface Proposal

The future CLI should look like:

- `hookr gen`
- `hookr build`
- `hookr inspect`
- `hookr call`
- `hookr bench`
- `hookr tui`

Expected details:

### `hookr inspect`

Current:

- schema-only

Planned:

- schema-only inspection
- schema plus Wasm inspection
- optional JSON output

### `hookr call`

Planned flags:

- `--schema`
- `--wasm`
- `--method`
- `--input`
- `--host-fixture`
- `--output`
- `--json`

### `hookr tui`

Planned flags:

- `--schema`
- `--wasm`
- `--host-fixture`

## Example Coverage

The first-party example contracts should double as fixtures for the new
developer tooling:

- `urlbalancer`
  - exercises plugin calls plus host RNG callbacks
- `textfilter`
  - exercises a simpler single-call contract
- `tickloop`
  - exercises tight-loop and benchmarking-oriented behavior

Each example should eventually include:

- request fixtures
- host callback fixtures when needed
- expected JSON outputs
- docs showing the related `hookr inspect` and `hookr call` flows

## Testing Strategy

This work should land with strong automated coverage.

Test layers:

- unit tests for reflection-driven JSON/FlatBuffers conversion
- unit tests for callback fixture playback
- integration tests for `hookr inspect --wasm`
- integration tests for `hookr call`
- fixture-based golden tests for JSON input and output
- end-to-end tests over the example contracts

Critical cases:

- missing required methods
- missing optional methods
- schema hash mismatch
- malformed request JSON
- unsupported field shapes
- callback fixture exhaustion
- callback fixture method mismatch
- plugin traps and normal error returns

## Risks

### Reflection conversion complexity

JSON to FlatBuffers conversion is the main complexity point. The plan should
start with the subset used by current example contracts, then expand carefully.

### TUI scope creep

The TUI is valuable, but it should not delay the CLI path. The CLI must remain
the execution engine, and the TUI should sit on top.

### Confusing the runtime and the dev harness

The generated SDK remains the normal integration path. The new tooling is a dev
and debugging layer, not the recommended production host API.

## Definition Of Done

This loop is closed when Hookr can:

- generate and build a plugin
- inspect a plugin module against its contract
- invoke plugin methods from the CLI
- stub host callbacks generically
- save and replay request fixtures
- provide an interactive schema-aware TUI

At that point, Hookr will support the full development loop for plugin authors
and host authors, not just code generation and runtime embedding.
