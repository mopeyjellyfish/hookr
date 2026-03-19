# Plugin Debugging And Development Tooling

Hookr can already generate bindings, build plugins, inspect schema-derived
contract metadata, make ad hoc calls, stub host callbacks, run benchmarks, and
explore a plugin through a TUI. This document records the reasoning behind that
developer workflow and the smaller improvements that still remain.

## Problem

Today Hookr is strong at:

- generating Go host SDK and plugin PDK glue from FlatBuffers
- building standard-Go `wasip1/wasm` plugin binaries
- inspecting a schema or a schema + plugin pair from the CLI
- making ad hoc calls into a plugin with JSON fixtures
- supplying generic host callback fixtures
- exploring plugin methods through the Bubble Tea TUI
- benchmarking fixture contracts

Today Hookr is still weaker at:

- publishing broader benchmark/debug snapshots from the tooling
- surfacing richer callback traces and replay artifacts
- giving developers more packaged workflows around iterative fixture capture
- turning the current TUI into a more advanced performance and trace console

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
  --plugin ./plugin.wasm
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
  --plugin ./plugin.wasm \
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
  --plugin ./plugin.wasm \
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
  --plugin ./plugin.wasm
```

Expected behavior:

- browse plugin methods
- inspect request and response fields
- edit request JSON
- invoke methods repeatedly
- see callback activity and validation errors
- save fixtures from the session

## Current Status

The main loop is now implemented:

- `hookr inspect --schema ... --plugin ...`
- `hookr call --schema ... --plugin ... --method ...`
- `hookr call ... --host-fixture ...`
- `hookr tui --schema ... --plugin ...`

The underlying design also landed the way this plan intended:

- schema plus plugin is the unit of inspection
- FlatBuffers remains the runtime transport
- JSON remains the human-facing format for CLI/TUI workflows
- the TUI sits on top of the same execution engine used by `hookr call`
- callback fixtures are contract-driven and method-oriented

Host fixture keys now use `Service.Method`, for example:

```json
{
  "Rng.Int": { "response": { "value": 1 } },
  "Rng.Float": { "response": { "value": 0.5 } }
}
```

## Remaining Improvements

The tooling loop is closed, but there is still room to improve it.

### Better Trace And Snapshot Workflows

- add optional callback trace output for repeated CLI/TUI debugging sessions
- add easier response snapshot export for golden-style debugging
- standardize how traces and snapshots are stored beside fixtures

### Richer TUI Diagnostics

- make loop and callback statistics more comprehensive
- add clearer callback activity summaries when a plugin is callback-heavy
- keep improving readability for long responses and runtime metadata

### Broader Fixture Coverage

- add more checked-in fixture flows for callback-heavy modules
- add more examples that exercise optional methods and failure modes
- align benchmark fixtures with the same reproducible request/host-fixture
  workflows

## Architecture Changes Needed

### Reuse and extend existing packages

The current repo already has useful building blocks:

- `internal/contract`
- `internal/inspect`
- `internal/flatc`
- `runtime`

The plan should extend those pieces rather than introduce parallel paths.

### Packages used today

- `internal/call`
  - coordinates schema loading, plugin loading, invocation, and output
- `internal/devhost`
  - generic callback stubbing and fixture playback
- `internal/tui`
  - Bubble Tea interface over the same call/session engine

The exact names can change, but the separation should be maintained.

### Reflection-driven bridge

The core enabling layer is a reflection-driven bridge between:

- JSON-friendly request/response values
- FlatBuffers request/response payloads

That bridge is the main technical requirement for both `hookr call` and the
future TUI.

## Command Surface Proposal

The current CLI now looks like:

- `hookr gen`
- `hookr build`
- `hookr inspect`
- `hookr call`
- `hookr bench`
- `hookr tui`

The main remaining CLI/TUI work is refinement, not missing core commands.

## Example Coverage

The first-party example contracts should double as fixtures for the new
developer tooling:

- `urlbalancer`
  - exercises plugin calls plus host RNG callbacks
- `textfilter`
  - exercises a simpler single-call contract
- `tickloop`
  - exercises tight-loop and benchmarking-oriented behavior

Each example should continue to include:

- request fixtures
- host callback fixtures when needed
- expected JSON outputs
- docs showing the related `hookr inspect` and `hookr call` flows

## Testing Strategy

This work should land with strong automated coverage.

Test layers:

- unit tests for reflection-driven JSON/FlatBuffers conversion
- unit tests for callback fixture playback
- integration tests for `hookr inspect --plugin`
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

The original loop described in this plan is now closed:

- Hookr can generate and build a plugin
- Hookr can inspect a plugin module against its contract
- Hookr can invoke plugin methods from the CLI
- Hookr can stub host callbacks generically
- Hookr can save and replay request fixtures
- Hookr provides an interactive schema-aware TUI

What remains is polish, better traces, and broader benchmark/debug coverage,
not a missing developer workflow.
