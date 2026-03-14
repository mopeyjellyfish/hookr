# Generated Go API Reference

`hookr gen` produces two primary generated surfaces for each contract package.

## Host SDK

Typical host API:

- `Open(ctx, Config) (*Runtime, error)`
- `(*Runtime).Close(ctx) error`
- typed plugin methods, for example `Balance(ctx, *BalanceRequestT) (*BalanceResponseT, error)`
- `SupportsX()` optional-method introspection helpers

`Config` includes:

- `WasmPath`
- `FileOptions` for trust policy, such as `WithHash(...)` or `WithAllowUnsigned()`
- `Host` (generated host callback interface)
- `RuntimeOptions` (forwarded to runtime)

If the contract defines no host callbacks, the generated `Config` will omit
`Host`.

## Plugin PDK

Typical plugin API:

- generated `Plugin` interface
- `RegisterPlugin(plugin)` / `MustRegisterPlugin(plugin)`
- `PluginContext` callback helpers, for example `ctx.RngInt(req)`

## Generated Files

For a package `mycontracthookr`, expect:

- FlatBuffers generated type files (`flatc` output)
- `contract_meta_gen.go`
- `host_sdk_gen.go`
- `plugin_pdk_gen.go`

See real generated examples:

- [urlbalancer generated package](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/urlbalancer/gen/urlbalancerhookr)
- [tickloop generated package](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/tickloop/gen/tickloophookr)
