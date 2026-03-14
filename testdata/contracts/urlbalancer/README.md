# UrlBalancer Contract

`urlbalancer` is the first intended end-to-end Hookr example contract.

It is deliberately consumer-defined. The methods in this schema are not Hookr
concepts; they model a host application's plugin API.

For a full step-by-step host and plugin walkthrough, see
[`docs/tutorials/urlbalancer.md`](../../../docs/tutorials/urlbalancer.md).

The contract demonstrates:

- host-defined plugin methods,
- structured request/response payloads,
- plugin -> host callbacks,
- a small, understandable end-to-end fixture for tests and docs.

## Contract Summary

Plugin methods:

- `GetInfo`
- `Balance`

Host callbacks:

- `RngInt`
- `RngFloat`

`Balance` accepts:

- a URL to validate,
- a list of backend nodes the plugin may route to.

The plugin is expected to:

1. validate and normalize the URL,
2. ask the host for RNG values,
3. choose one backend node,
4. return the selected node and validation details.

## Why This Fixture Exists

This example proves the Hookr shape without baking application semantics into
Hookr itself. DICE will later define its own FlatBuffers contract in the same
way.

## Working Flow

Generate the contract glue:

```bash
hookr gen \
  --schema ./testdata/contracts/urlbalancer/urlbalancer.fbs \
  --out ./testdata/contracts/urlbalancer/gen \
  --package urlbalancerhookr
```

Build the plugin:

```bash
hookr build \
  --plugin ./testdata/contracts/urlbalancer/plugin \
  --out ./testdata/contracts/urlbalancer/bin/urlbalancer.wasm
```

Inspect the contract:

```bash
hookr inspect \
  --schema ./testdata/contracts/urlbalancer/urlbalancer.fbs \
  --package urlbalancerhookr
```

The generated package lives under `testdata/contracts/urlbalancer/gen/urlbalancerhookr`.
The fixture also includes:

- `plugin/main.go`: TinyGo plugin implementation
- `examples/host_main.go`: host-side example
- `e2e_test.go`: end-to-end verification
