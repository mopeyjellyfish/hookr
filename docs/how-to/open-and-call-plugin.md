# Open And Call A Plugin Runtime

## Goal

Load a plugin Wasm module and call generated typed methods from a Go host.

## Steps

1. Implement generated host callback interface.

2. Open runtime through generated package:

```go
rt, err := mycontracthookr.Open(ctx, mycontracthookr.Config{
	WasmPath: "./bin/plugin.wasm",
	Host:     hostImpl,
	FileOptions: []hookrruntime.FileOption{
		hookrruntime.WithAllowUnsigned(),
	},
})
if err != nil {
	return err
}
defer rt.Close(ctx)
```

For production plugins, replace `WithAllowUnsigned()` with a pinned hash or
custom verification policy.

3. Call generated plugin methods:

```go
resp, err := rt.SomeMethod(ctx, &mycontracthookr.SomeRequestT{
	// request fields
})
if err != nil {
	return err
}
_ = resp
```

## Contract Validation

`Open` configures Hookr runtime contract checks using generated schema metadata.
Schema mismatch and missing required methods fail at startup.

## Related

- [Reference: ABI](../reference/abi.md)
- [Reference: Generated Go APIs](../reference/generated-go-api.md)
