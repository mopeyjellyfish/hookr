# Open And Call A Plugin Runtime

## Goal

Load a plugin Wasm module and call generated typed methods from a Go host.
If your contract defines `rpc_service Host`, bind those callbacks so plugins can
call back into the host.

## Steps

1. Implement generated host callback interface.

The generated package tells you exactly what to implement. If the schema
defines:

```fbs
rpc_service Host {
  RngInt(RngIntRequest):RngIntResponse;
}
```

the generated Go package will expose:

```go
type Host interface {
	RngInt(context.Context, *mycontracthookr.RngIntRequestT) (*mycontracthookr.RngIntResponseT, error)
}
```

2. Open runtime through generated package:

```go
	plugin, err := mycontracthookr.Open(ctx, mycontracthookr.Config{
		PluginPath: "./bin/plugin.wasm",
	Host:     host{},
	FileOptions: []hookr.FileOption{
		hookr.WithAllowUnsigned(),
	},
})
if err != nil {
	return err
}
defer plugin.Close(ctx)
```

For production plugins, replace `WithAllowUnsigned()` with a pinned hash or
custom verification policy.

`PluginPath` is the plugin artifact path for this contract. You can point it at
any `.wasm` built against the same generated schema package.

3. Call generated plugin methods:

```go
resp, err := plugin.SomeMethod(ctx, &mycontracthookr.SomeRequestT{
	// request fields
})
if err != nil {
	return err
}
_ = resp
```

For contracts with host callbacks, `host{}` must implement the generated `Host`
interface. A minimal example looks like this:

```go
type host struct{}

func (host) RngInt(ctx context.Context, req *mycontracthookr.RngIntRequestT) (*mycontracthookr.RngIntResponseT, error) {
	return &mycontracthookr.RngIntResponseT{Value: req.Min}, nil
}
```

4. Use the callbacks from plugin code through `PluginContext`:

```go
func (plugin) Balance(ctx *mycontracthookr.PluginContext, req *mycontracthookr.BalanceRequestT) (*mycontracthookr.BalanceResponseT, error) {
	rng, err := ctx.RngInt(&mycontracthookr.RngIntRequestT{Min: 0, Max: 3})
	if err != nil {
		return nil, err
	}
	return &mycontracthookr.BalanceResponseT{Slot: rng.Value}, nil
}
```

There is no separate registration step for host methods. Passing `host{}` into
`Config.Host` is the registration step.

## Contract Validation

`Open` configures Hookr runtime contract checks using generated schema metadata.
Schema mismatch and missing required methods fail at startup.

## Related

- [Reference: ABI](../reference/abi.md)
- [Reference: Generated Go APIs](../reference/generated-go-api.md)
