# Contract Samples

This directory contains schema-first contract examples for ABI v2 development.

- `greeter/greeter.capnp`: sample Cap'n Proto schema
- `greeter/greeter.proto`: sample Protobuf schema
- `greeter/contract.json`: optional manifest override for `hookr-gen`
- `greeter/gen/*.go`: generated metadata + runtime/PDK glue stubs
- `greeter/gen-proto/*.go`: generated metadata + runtime/PDK glue stubs from protobuf
- `greeter/examples/*`: host/plugin reference wiring + generated file index

Generated glue now includes switch-based direct-dispatch helpers:

- Host side: `RuntimeCallHandlerV2(...)`
- Plugin side: `SetPluginMethodDispatcher(...)`

Generated packages also expose:

- `ContractCapabilities` + per-method capability constants
- typed wrappers (`BindRuntime*`, `Host*`, `CallPlugin*`, `BindPlugin*`, `RegisterPlugin*`, `CallHost*`)

Generate the sample files with:

```bash
mkdir -p .cache/go-build
GOCACHE=$(pwd)/.cache/go-build go run ./cmd/hookr-gen \
  -schema ./testdata/contracts/greeter/greeter.capnp \
  -manifest ./testdata/contracts/greeter/contract.json \
  -out ./testdata/contracts/greeter/gen \
  -package greetercontract \
  -codec capnp
```

You can also generate directly from protobuf service definitions (no manifest required):

```bash
mkdir -p .cache/go-build
GOCACHE=$(pwd)/.cache/go-build go run ./cmd/hookr-gen \
  -schema ./testdata/contracts/greeter/greeter.proto \
  -service Greeter \
  -out ./testdata/contracts/greeter/gen \
  -package greetercontract \
  -codec protobuf
```
