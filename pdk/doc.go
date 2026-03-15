/*
Package pdk provides the low-level plugin runtime primitives used by generated Hookr
plugin bindings.

Normal Hookr plugin authors should not build against this package directly. The
intended workflow is:

1. Define the application contract in FlatBuffers.
2. Run `hookr gen`.
3. Implement the generated plugin interface.
4. Export `hookr_init` and call the generated `MustRegisterPlugin(...)` helper.

For example:

	package main

	import "github.com/acme/example/gen/examplehookr"

	type Plugin struct{}

	func (Plugin) Balance(
		ctx *examplehookr.PluginContext,
		req *examplehookr.BalanceRequestT,
	) (*examplehookr.BalanceResponseT, error) {
		return &examplehookr.BalanceResponseT{}, nil
	}

	//go:wasmexport hookr_init
	func hookrInit() {
		examplehookr.MustRegisterPlugin(Plugin{})
	}

The pdk package exists so generated code can:

- publish Hookr ABI handshake metadata,
- dispatch host calls by numeric method ID,
- exchange request and response buffers with the host runtime,
- surface plugin errors through the Hookr ABI.

Advanced users can use this package directly, but it is primarily an
implementation dependency of generated plugin code rather than the recommended
application-facing API.
*/
package pdk
