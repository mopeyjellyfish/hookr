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

- `PluginPath` for the plugin artifact path; swap in any `.wasm` built for the same contract
- `FileOptions` for trust policy, such as `WithHash(...)` or `WithAllowUnsigned()`
- `Host` (generated aggregate struct containing host modules)
- `RuntimeOptions` (forwarded to runtime)

If the contract defines no host callbacks, the generated `Config` will omit
`Host`.

### Host Callback Story

Host callbacks come from every non-`Plugin` `rpc_service` in the schema. Hookr
auto-discovers those services as host modules.

For a contract like:

```fbs
rpc_service Rng {
  Int(RngIntRequest):RngIntResponse;
  Float(RngFloatRequest):RngFloatResponse;
}
```

Hookr generates:

- a `Host` aggregate struct in the generated package
- one interface per host module, such as `RngHost`
- binding code used by `Open(...)` to expose those methods to the plugin

That means the host writes normal Go methods and passes the implementation into
`Config.Host`:

```go
type host struct{}

func (host) Int(ctx context.Context, req *mycontracthookr.RngIntRequestT) (*mycontracthookr.RngIntResponseT, error) {
	return &mycontracthookr.RngIntResponseT{Value: req.Min}, nil
}

plugin, err := mycontracthookr.Open(ctx, mycontracthookr.Config{
	PluginPath: "./plugin.wasm",
	Host: mycontracthookr.Host{
		Rng: host{},
	},
})
```

The host does not manually register callback names or method IDs.

## Plugin PDK

Typical plugin API:

- generated `Plugin` interface
- `RegisterPlugin(plugin)` / `MustRegisterPlugin(plugin)`
- module clients on `PluginContext`, for example `ctx.Rng.Int(req)`
- borrowed-view host callback helpers for hot paths, for example `ctx.Rng.IntView(req, func(*RngIntResponse) error) error`

### Plugin Use Of Host Callbacks

Inside plugin code, generated `PluginContext` exposes one helper per host
callback method, grouped by service:

```go
func (plugin) Balance(ctx *mycontracthookr.PluginContext, req *mycontracthookr.BalanceRequestT) (*mycontracthookr.BalanceResponseT, error) {
	rng, err := ctx.Rng.Int(&mycontracthookr.RngIntRequestT{Min: 0, Max: 3})
	if err != nil {
		return nil, err
	}
	return &mycontracthookr.BalanceResponseT{Slot: rng.Value}, nil
}
```

So the callback path is:

1. schema defines host modules as non-`Plugin` `rpc_service`s
2. host implements generated module interfaces
3. `Open(...)` binds that implementation
4. plugin calls generated module clients on `PluginContext`

## Generated Files

For a package `mycontracthookr`, expect:

- FlatBuffers generated type files (`flatc` output)
- `contract_meta_gen.go`
- `host_sdk_gen.go`
- `plugin_pdk_gen.go`

See real generated examples:

- [urlbalancer generated package](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/urlbalancer/gen/urlbalancerhookr)
- [tickloop generated package](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/tickloop/gen/tickloophookr)
