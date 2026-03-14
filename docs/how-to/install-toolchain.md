# Install Hookr Toolchain

## Goal

Install the required CLI and compiler tools for Hookr development.

## Steps

1. Install Hookr:

```bash
go install github.com/mopeyjellyfish/hookr/cmd/hookr@latest
```

2. Install FlatBuffers compiler (`flatc`) and TinyGo:

```bash
brew install flatbuffers tinygo
```

3. Verify tools:

```bash
hookr --help
flatc --version
tinygo version
```

## Related

- [Generate Glue From A Contract](./generate-glue.md)
- [Build A Plugin Artifact](./build-plugin.md)
