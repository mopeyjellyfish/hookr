# Install Hookr Toolchain

## Goal

Install the required CLI and compiler tools for Hookr development.

## Steps

1. Install Go 1.26 or higher.

2. Install Hookr:

```bash
go install github.com/mopeyjellyfish/hookr/cmd/hookr@latest
```

3. Install FlatBuffers compiler (`flatc`) and TinyGo 0.41.0 or higher:

```bash
brew install flatbuffers tinygo
```

4. Verify tools:

```bash
hookr --help
flatc --version
tinygo version
```

## Related

- [Generate Glue From A Contract](./generate-glue.md)
- [Build A Plugin Artifact](./build-plugin.md)
