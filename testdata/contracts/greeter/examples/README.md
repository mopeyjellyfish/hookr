# Greeter ABI v2 Examples

This folder is a quick index for people evaluating Hookr contract generation.

## Generated Contract Packages To Inspect

Cap'n Proto generated output:

- `testdata/contracts/greeter/gen/contract_meta_gen.go`
- `testdata/contracts/greeter/gen/runtime_glue_gen.go`
- `testdata/contracts/greeter/gen/pdk_glue_gen.go`

Protobuf generated output:

- `testdata/contracts/greeter/gen-proto/contract_meta_gen.go`
- `testdata/contracts/greeter/gen-proto/runtime_glue_gen.go`
- `testdata/contracts/greeter/gen-proto/pdk_glue_gen.go`

## Reference Host/Plugin Wiring

- Host example: `testdata/contracts/greeter/examples/host_main.go`
- Plugin example: `testdata/contracts/greeter/examples/plugin_main.go`

Both examples are tagged with `//go:build ignore` so they are not built by CI.
