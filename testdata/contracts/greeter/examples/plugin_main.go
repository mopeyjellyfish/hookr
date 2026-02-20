//go:build ignore

package main

import (
	"fmt"

	"github.com/mopeyjellyfish/hookr/pdk"
	greetercontract "github.com/mopeyjellyfish/hookr/testdata/contracts/greeter/gen"
)

//go:wasmexport hookr_init
func Initialize() {
	// Publish ABI v2 handshake data (schema hash + ABI version) for host validation.
	greetercontract.EnablePDKHandshake()

	// Fast-path ABI v2 plugin dispatcher (switch-based, no map lookup).
	greetercontract.SetPluginMethodDispatcher(greetercontract.PluginMethodHandlers{
		MethodHello: func(payload []byte) ([]byte, error) {
			// In generated typed stubs, payload decode/encode is generated for your schema types.
			return []byte("hello-from-plugin"), nil
		},
	})

	// Optional fallback registration path (map-based):
	// pdk.FnMethod(greetercontract.MethodHello, func(payload []byte) ([]byte, error) { ... })
}

func callHost(payload []byte) ([]byte, error) {
	resp, err := pdk.HostCallMethod(greetercontract.MethodHello, payload)
	if err != nil {
		return nil, fmt.Errorf("host call failed: %w", err)
	}
	return resp, nil
}
