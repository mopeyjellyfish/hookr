# Generate Glue From A Contract

## Goal

Generate FlatBuffers bindings plus Hookr SDK/PDK glue from a schema.

## Steps

1. Run `hookr gen`:

```bash
hookr gen \
  --schema ./contract.fbs \
  --out ./gen \
  --package mycontracthookr
```

2. For imported schemas, add include paths:

```bash
hookr gen \
  --schema ./schemas/contract.fbs \
  --out ./gen \
  --package mycontracthookr \
  --include ./schemas \
  --include ./third_party
```

3. For non-default plugin service names:

```bash
hookr gen \
  --schema ./contract.fbs \
  --out ./gen \
  --package mycontracthookr \
  --plugin-service EnginePlugin
```

Every `rpc_service` other than the configured plugin service is
auto-discovered as a host callback module.

4. To generate Rust plugin-side bindings instead of the Go SDK/PDK package:

```bash
hookr gen \
  --lang rust \
  --schema ./contract.fbs \
  --out ./gen \
  --package mycontracthookr
```

The Go host SDK remains the host integration path. The Rust output is for
building a plugin Wasm module that implements the same Hookr ABI. The plugin
crate still needs to export `hookr_init` and call
`hookr_plugin::register_plugin(...)` with its implementation.

5. To generate Zig plugin-side ABI bindings:

```bash
hookr gen \
  --lang zig \
  --schema ./contract.fbs \
  --out ./gen \
  --package mycontracthookr
```

The Zig output provides the Hookr ABI exports/imports, method constants,
schema hash, capabilities, and raw host callback transport helpers. Typed
FlatBuffers ergonomics are expected to build on top of that ABI layer.

6. To generate C++ plugin-side bindings:

```bash
hookr gen \
  --lang cpp \
  --schema ./contract.fbs \
  --out ./gen \
  --package mycontracthookr
```

The C++ output combines FlatBuffers C++ headers with a Hookr ABI header for
building plugin Wasm modules that the Go host SDK can load.

## Output

Generated Go packages typically include:

- FlatBuffers type files (`flatc` output)
- `contract_meta_gen.go`
- `host_sdk_gen.go`
- `plugin_pdk_gen.go`

Generated Rust plugin packages typically include:

- FlatBuffers Rust type files (`flatc --rust` output)
- `lib.rs`
- `hookr_plugin.rs`

Generated Zig plugin packages typically include:

- `hookr_plugin.zig`

Generated C++ plugin packages typically include:

- FlatBuffers C++ headers (`flatc --cpp` output)
- `hookr_plugin.hpp`

## Related

- [Reference: CLI](../reference/cli.md)
- [Reference: Contract Model](../reference/contracts.md)
