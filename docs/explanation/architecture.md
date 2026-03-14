# Architecture Overview

Hookr is intentionally split into two responsibilities:

1. application-defined contract surface
2. Hookr-owned runtime transport and generation

## What The Host Application Owns

- FlatBuffers schema (`.fbs`)
- plugin service methods
- host callback service methods
- implementation logic for host and plugin

## What Hookr Owns

- Wasm ABI and host module bindings
- handshake and compatibility validation
- contract loading and canonical schema hashing
- generated host SDK and plugin PDK glue
- CLI orchestration (`gen`, `build`, `inspect`, `bench`)

## Why This Split

- keeps Hookr generic across applications
- avoids framework-level opinionation about game/app methods
- allows generated code to stay typed and low-overhead
- keeps serialization logic in official FlatBuffers tooling

For full system direction, see:
[Hookr Plugin System](../plugin-system.md).
