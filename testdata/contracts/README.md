# Contract Samples

This directory contains Hookr contract fixtures.

Primary FlatBuffers fixtures:

- `urlbalancer/`: plugin -> host callback example with generated Go SDK/PDK and e2e test
- `textfilter/`: minimal no-callback tutorial-style example with generated Go SDK/PDK and e2e test
- `tickloop/`: hot-loop benchmark example with optional plugin method, generated Go SDK/PDK, e2e test, and benchmarks

Generate a FlatBuffers fixture package with:

```bash
hookr gen \
  --schema ./testdata/contracts/urlbalancer/urlbalancer.fbs \
  --out ./testdata/contracts/urlbalancer/gen \
  --package urlbalancerhookr
```

Build a plugin with:

```bash
hookr build \
  --plugin ./testdata/contracts/urlbalancer/plugin \
  --out ./testdata/contracts/urlbalancer/bin/urlbalancer.wasm
```

Inspect a contract with:

```bash
hookr inspect \
  --schema ./testdata/contracts/tickloop/tickloop.fbs \
  --package tickloophookr
```

Run the benchmark fixture with:

```bash
hookr bench \
  --package ./testdata/contracts/tickloop
```
