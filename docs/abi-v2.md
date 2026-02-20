# Hookr ABI v2 (Draft)

This document defines the first schema-driven ABI for Hookr that targets:

- one boundary copy per direction (host <-> guest memory),
- zero-copy decoding on each side where the codec supports it (for example Cap'n Proto),
- minimal dispatch overhead (numeric method IDs instead of string lookup in hot paths),
- generated glue code for host/plugin contracts.

## Goals

1. Keep the runtime and PDK generic; plugin contracts are user-defined.
2. Allow generated stubs to avoid reflection and dynamic type assertions.
3. Ensure contract compatibility is verified before the first business call.
4. Preserve backward compatibility with existing byte/msgpack APIs.

## Non-goals

1. True shared-memory zero-copy between host and guest. Wasm 1.0 has separate linear memories.
2. Forcing one codec. ABI v2 supports codec-specific generated stubs.

## Terms

- Contract: the schema-defined API for a plugin module.
- Method ID: numeric operation identifier generated from schema.
- Schema hash: fixed digest identifying an exact contract version.
- Handshake: ABI version + schema hash exchange used for compatibility checks.

## Handshake

Both sides expose the same handshake payload:

- `abi_major` (uint16)
- `abi_minor` (uint16)
- `schema_hash` (32 bytes)
- `capabilities` (uint64 bitmask)

Compatibility rules:

1. `abi_major` must match exactly.
2. `schema_hash` must match exactly.
3. `capabilities`: plugin must include all host-required capability bits.
4. `abi_minor` may differ; host may accept lower plugin minor versions if features are not required.

## Dispatch Model

ABI v2 call dispatch is method-ID based:

- Host -> plugin: `method_id:uint32`, request bytes.
- Plugin -> host callback: `method_id:uint32`, request bytes.

String operation names remain a compatibility layer only.

For highest throughput, generated code can install direct switch-based dispatchers:

1. Host side: `RuntimeCallHandlerV2(...)` + `runtime.WithCallHandlerV2(...)`
2. Plugin side: `SetPluginMethodDispatcher(...)`

Method ID source rules in the current generator:

1. Cap'n Proto: use explicit method IDs from the schema (`method @<id>`).
2. Protobuf: derive stable IDs from `FNV-1a(service + "." + method)`.

## Generated Glue

Codegen should emit:

1. Method ID constants.
2. Contract metadata (method table + schema hash).
3. Host-side typed client wrappers (method ID + codec encode/decode).
4. Plugin-side registration adapters (method ID + codec decode/encode).
5. Typed helper wrappers for binding handlers/callers without reflection.

Generated code owns contract-specific serialization details and should not rely on reflection.

## Memory and Copy Boundaries

Expected steady-state copy behavior:

1. Host copies request into guest memory.
2. Plugin decodes request from its linear memory bytes.
3. Plugin encodes response into guest memory.
4. Host copies response out of guest memory.

No additional copies are required in glue code beyond codec/runtime requirements.

## Rollout Plan

1. Land contract primitives (`runtime/contract`, `pdk/contract`) and this ABI spec.
2. Add `hookr-gen` to generate method tables and typed stubs from user schema.
3. Add runtime/PDK v2 invoke path keyed by method ID.
4. Keep existing APIs (`PluginFnByte`, `PluginFnSerial`, `FnByte`, `FnSerial`) for compatibility.
