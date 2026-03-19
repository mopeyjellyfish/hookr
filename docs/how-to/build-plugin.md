# Build A Plugin Artifact

## Goal

Compile a plugin package into a Wasm module with Hookr defaults.

## Steps

1. Build plugin:

```bash
hookr build \
  --plugin ./plugin \
  --out ./bin/plugin.wasm
```

2. Optional: override the Go binary or build mode:

```bash
hookr build \
  --plugin ./plugin \
  --out ./bin/plugin.wasm \
  --go /path/to/go \
  --buildmode c-shared
```

## Notes

- Hookr builds plugins with `go build` under `GOOS=wasip1 GOARCH=wasm`.
- The default build mode is `c-shared`, which produces a WASI reactor with `_initialize`.
- Plugin packages should use `package main`, export `hookr_init`, and define an empty `main()`.

## Related

- [Reference: CLI](../reference/cli.md)
- [How To Open And Call A Plugin Runtime](./open-and-call-plugin.md)
