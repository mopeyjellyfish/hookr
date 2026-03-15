# Hookr ABI

This document belongs to the Diataxis Reference section:
[Reference Index](./reference/index.md).

This document describes the Hookr method-ID ABI used by the schema-defined
FlatBuffers plugin system.

The ABI is intentionally small. Hookr-generated code owns contract-specific
FlatBuffers encoding and decoding, while the runtime and PDK own transport,
validation, and dispatch.

## Goals

1. Keep the runtime and PDK generic while host applications define their own plugin methods.
2. Make hot-path dispatch numeric and predictable.
3. Validate compatibility before the first business call.
4. Keep the ABI stable enough for future non-Go PDKs and SDKs.

## Non-goals

1. Shared native pointers between host and guest. Wasm linear memory is the boundary.
2. Application semantics. Hookr transports methods like `Balance`, `Tick`, or `Validate`; it does not define them.
3. Zero copies everywhere. The practical target is one boundary copy per direction with zero-copy reads where buffer ownership allows it.

## Terms

- Contract: the FlatBuffers-defined API for a plugin module.
- Method ID: numeric operation identifier derived from the contract.
- Schema hash: fixed digest identifying the exact normalized contract shape.
- Handshake: ABI version, schema hash, capabilities, and method metadata exchange used for compatibility checks.

## Handshake

Both sides expose the same handshake payload:

- `abi_major` (`uint16`)
- `abi_minor` (`uint16`)
- `schema_hash` (`[32]byte`)
- `capabilities` (`uint64`)

Generated plugins also publish method metadata so the host can distinguish
required methods from optional ones before first call.

Compatibility rules:

1. `abi_major` must match exactly.
2. `abi_minor` must match exactly.
3. `schema_hash` must match exactly.
4. The plugin must include all host-required capability bits.

## Dispatch Model

Call dispatch is method-ID based:

- host to plugin: `method_id:uint32`, request bytes
- plugin to host callback: `method_id:uint32`, request bytes

Generated code installs direct switch-based dispatchers on both sides:

1. Host side: generated host callbacks wired through `runtime.WithHostMethodFns(...)`
2. Plugin side: generated plugin dispatcher wired through the PDK

The public Go API hides raw method IDs behind generated wrappers.

## Generated Glue

`hookr gen` emits:

1. method ID constants
2. contract metadata, including the canonical schema hash
3. host-side typed client wrappers
4. plugin-side registration adapters
5. typed host-callback helpers on `PluginContext`
6. optional-method introspection helpers for host code

Generated code owns FlatBuffers-specific serialization logic and should not rely
on reflection in steady-state call paths.

## Memory and Copy Boundaries

Expected steady-state behavior:

1. Host copies the request into guest memory.
2. Plugin reads request data from guest linear memory.
3. Plugin writes the response into guest memory.
4. Host copies the response out of guest memory.

This is the realistic fast path for Wasm today. Hookr aims to minimize
additional copies and allocations above that baseline.

## Safety Properties

The runtime should:

- reject incompatible schema hashes before first business call
- fail closed on malformed guest pointers instead of panicking the host process
- surface bootstrap and callback failures as normal errors
- require explicit trust for unsigned plugin artifacts

## Evolution

The ABI is stable Hookr surface, not a parallel feature track. Future work
should improve generated bindings, benchmarks, and language backends without
forking the runtime contract.
