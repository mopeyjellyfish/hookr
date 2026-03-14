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

2. Optional: override TinyGo binary and build settings:

```bash
hookr build \
  --plugin ./plugin \
  --out ./bin/plugin.wasm \
  --tinygo tinygo \
  --target wasip1 \
  --buildmode c-shared \
  --scheduler none \
  --no-debug
```

## Notes

- Hookr uses TinyGo for plugin builds.
- Defaults are tuned for runtime plugin loading (`wasip1`, `c-shared`).

## Related

- [Reference: CLI](../reference/cli.md)
- [How To Open And Call A Plugin Runtime](./open-and-call-plugin.md)
