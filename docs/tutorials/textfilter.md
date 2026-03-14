# Build A Minimal Plugin With TextFilter Fixture

This tutorial is the smallest end-to-end Hookr path: host-to-plugin calls with
no host callbacks.

Use it when you want to learn Hookr fundamentals before adding callback
services.

## Steps

1. Read the fixture contract:
   [TextFilter Contract](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/textfilter)
2. Generate code:

```bash
hookr gen \
  --schema ./testdata/contracts/textfilter/textfilter.fbs \
  --out ./testdata/contracts/textfilter/gen \
  --package textfilterhookr
```

3. Build plugin Wasm:

```bash
hookr build \
  --plugin ./testdata/contracts/textfilter/plugin \
  --out ./testdata/contracts/textfilter/bin/textfilter.wasm
```

4. Run fixture tests:

```bash
go test ./testdata/contracts/textfilter -count=1
```

Then continue with callback-driven integrations in:
[Build A Host And Plugin With FlatBuffers](./urlbalancer.md).
