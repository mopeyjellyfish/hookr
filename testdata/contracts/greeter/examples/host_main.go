//go:build ignore

package main

import (
	"context"
	"log"

	"github.com/mopeyjellyfish/hookr/runtime"
	greetercontract "github.com/mopeyjellyfish/hookr/testdata/contracts/greeter/gen"
)

func main() {
	ctx := context.Background()

	// Fast-path ABI v2 host callback dispatcher (switch-based, no map lookup).
	callHandler := greetercontract.RuntimeCallHandlerV2(greetercontract.RuntimeMethodHandlers{
		MethodHello: func(ctx context.Context, payload []byte) ([]byte, error) {
			// In generated typed stubs, payload decode/encode is generated for your schema types.
			return payload, nil
		},
	})

	rt, err := runtime.New(
		ctx,
		runtime.WithFile("./plugin.wasm"),
		runtime.WithCallHandlerV2(callHandler),
		// Verify ABI major + schema hash during startup.
		runtime.WithContractHandshake(greetercontract.RuntimeHandshake()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer rt.Close(ctx)

	hello, err := runtime.PluginFnMethod(rt, greetercontract.MethodHello)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := hello.Call(ctx, []byte("wire-encoded request"))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("plugin response len=%d", len(resp))
}
