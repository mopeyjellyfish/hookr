# Generated Go API Reference

`hookr gen` produces two primary generated surfaces for each contract package.

## Host SDK

Typical host API:

- `Open(ctx, Config) (*Runtime, error)`
- `(*Runtime).Close(ctx) error`
- typed plugin methods, for example `Balance(ctx, *BalanceRequestT) (*BalanceResponseT, error)`
- borrowed-view plugin methods for hot paths, for example `BalanceView(ctx, *BalanceRequestT, func(*BalanceResponse) error) error`
- `SupportsX()` optional-method introspection helpers

`Config` includes:

- `WasmPath` for the plugin artifact path; swap in any `.wasm` built for the same contract
- `FileOptions` for trust policy, such as `WithHash(...)` or `WithAllowUnsigned()`
- `Host` (generated host callback interface)
- `RuntimeOptions` (forwarded to runtime)

If the contract defines no host callbacks, the generated `Config` will omit
`Host`.

### Host Callback Story

Host callbacks come from `rpc_service Host` in the schema.

For a contract like:

```fbs
rpc_service Host {
  RngInt(RngIntRequest):RngIntResponse;
  RngFloat(RngFloatRequest):RngFloatResponse;
}
```

Hookr generates:

- a `Host` interface in the generated package
- one method on that interface per `Host` service method
- binding code used by `Open(...)` to expose those methods to the plugin

That means the host writes normal Go methods and passes the implementation into
`Config.Host`:

```go
type host struct{}

func (host) RngInt(ctx context.Context, req *mycontracthookr.RngIntRequestT) (*mycontracthookr.RngIntResponseT, error) {
	return &mycontracthookr.RngIntResponseT{Value: req.Min}, nil
}

plugin, err := mycontracthookr.Open(ctx, mycontracthookr.Config{
	WasmPath: "./plugin.wasm",
	Host:     host{},
})
```

The host does not manually register callback names or method IDs.

## Plugin PDK

Typical plugin API:

- generated `Plugin` interface
- `RegisterPlugin(plugin)` / `MustRegisterPlugin(plugin)`
- `PluginContext` callback helpers, for example `ctx.RngInt(req)`
- borrowed-view host callback helpers for hot paths, for example `ctx.RngIntView(req, func(*RngIntResponse) error) error`

### Plugin Use Of Host Callbacks

Inside plugin code, generated `PluginContext` exposes one helper per host
callback:

```go
func (plugin) Balance(ctx *mycontracthookr.PluginContext, req *mycontracthookr.BalanceRequestT) (*mycontracthookr.BalanceResponseT, error) {
	rng, err := ctx.RngInt(&mycontracthookr.RngIntRequestT{Min: 0, Max: 3})
	if err != nil {
		return nil, err
	}
	return &mycontracthookr.BalanceResponseT{Slot: rng.Value}, nil
}
```

So the callback path is:

1. schema defines `rpc_service Host`
2. host implements generated `Host` interface
3. `Open(...)` binds that implementation
4. plugin calls generated `PluginContext` helpers

## Generated Files

For a package `mycontracthookr`, expect:

- FlatBuffers generated type files (`flatc` output)
- `contract_meta_gen.go`
- `host_sdk_gen.go`
- `plugin_pdk_gen.go`

See real generated examples:

- [urlbalancer generated package](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/urlbalancer/gen/urlbalancerhookr)
- [tickloop generated package](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/tickloop/gen/tickloophookr)
