# CLI Reference

## Binary

`hookr`

Install it with:

```bash
go install github.com/mopeyjellyfish/hookr/cmd/hookr@latest
```

## Commands

### `hookr gen`

Generate FlatBuffers bindings and Hookr SDK/PDK glue.

Required flags:

- `--schema`
- `--out`
- `--package`

Key optional flags:

- `--flatc`
- `--include`, `-I`
- `--plugin-service`
- `--optional-attribute`
- `--capabilities`

### `hookr build`

Build a Hookr plugin Wasm module with TinyGo.

Required flags:

- `--plugin`
- `--out`

Optional flags:

- `--tinygo`
- `--target`
- `--buildmode`
- `--scheduler`
- `--no-debug`

### `hookr version`

Print the Hookr CLI version embedded in the binary.

### `hookr inspect`

Inspect Hookr contract metadata derived from a FlatBuffers schema, optionally
against a real plugin module.

Required flags:

- `--schema`

Key optional flags:

- `--plugin`
- `--host-fixture`
- `--hash`
- `--allow-unsigned`
- `--flatc`
- `--include`, `-I`
- `--package`
- `--plugin-service`
- `--optional-attribute`

### `hookr call`

Invoke a plugin method using schema-driven JSON input and output.

Required flags:

- `--schema`
- `--plugin`
- `--method`

Key optional flags:

- `--input`
- `--host-fixture`
- `--hash`
- `--allow-unsigned`
- `--flatc`
- `--include`, `-I`
- `--package`
- `--plugin-service`
- `--optional-attribute`

### `hookr tui`

Open an interactive Bubble Tea-based terminal UI for inspecting and invoking plugin methods.

Required flags:

- `--schema`
- `--plugin`

Key optional flags:

- `--host-fixture`
- `--hash`
- `--allow-unsigned`
- `--flatc`
- `--include`, `-I`
- `--package`
- `--plugin-service`
- `--optional-attribute`

Behavior notes:

- `inspect` requires explicit trust only when `--plugin` is provided
- `call` and `tui` always require explicit trust for the plugin artifact: pass `--hash` for pinned plugins or `--allow-unsigned` for local dev builds
- requests are pre-filled from the FlatBuffers schema
- the top bar shows the active schema, plugin, method, and live timing stats
- request text is read-only inside the UI; editing uses `$VISUAL`, then `$EDITOR`
- the plugin hot reloads when the plugin file changes on disk
- single-key actions support one-off calls, loop mode, request reset, and pretty formatting
- the shortcut legend stays visible at the bottom of the screen
- the debug pane shows handshake/runtime metadata and loop timing stats

### `hookr bench`

Run `go test` benchmarks with Hookr defaults.

Optional flags:

- `--package` (default: `./testdata/contracts/tickloop`)
- `--bench` (default: `.`)
- `--run` (default: `^$`)
- `--count` (default: `1`)

## Notes

- `hookr gen` expects FlatBuffers schemas and an available `flatc` binary.
- Hookr auto-discovers host callback modules from every `rpc_service` other than the configured plugin service.
- `hookr build` is TinyGo-first today; the CLI is structured so more build backends can be added later.
- `hookr inspect` is useful for confirming schema hash, method IDs, required versus optional methods, and handshake-visible metadata before you wire a host.
- `hookr call` is the fastest path for reproducing a request/response bug against a real plugin.
- `hookr tui` is a Bubble Tea UI over the same schema-driven call path used by `hookr call`.
- Commands keep machine-readable payloads on stdout and send progress/status messages to stderr.
