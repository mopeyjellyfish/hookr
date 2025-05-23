/*
Hookr is a high-performance WebAssembly plugin system that seamlessly bridges Go and WASM.

Hookr empowers Go applications with secure, type-safe WebAssembly modules,
enabling dynamic extensibility through a clean, bi-directional communication interface.
Develop modular applications with runtime-loadable components while maintaining
the performance and safety guarantees you expect.

# Basic Usage

The simplest way to use Hookr is to load a WASM plugin file and invoke a function:

	package main

	import (
	    "context"
	    "fmt"
	    "log"

	    "github.com/mopeyjellyfish/hookr/runtime"
	)

	func main() {
	    // Create a new plugin with the WASM file
	    ctx := context.Background()
	    rt, err := runtime.New(ctx, runtime.WithFile("./plugin.wasm"))
	    if err != nil {
	        log.Fatalf("Failed to load plugin: %v", err)
	    }
	    defer rt.Close(ctx)

	    // Invoke a function from the rt
	    result, err := rt.Invoke(ctx, "hello", []byte("world"))
	    if err != nil {
	        log.Fatalf("Failed to invoke function: %v", err)
	    }

	    fmt.Printf("Result: %s\n", result)
	}

# Security

Hookr provides security features such as hash verification to ensure the integrity
of loaded WASM modules:

	rt, err := runtime.New(ctx,
	    runtime.WithFile("./plugin.wasm",
	        runtime.WithHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
	        runtime.WithHasher(runtime.Sha256Hasher{}),
	    ),
	)

# Host Functions

Host functions can be registered to allow the plugin to call back into the runtime:

	hostFn := runtime.HostFn("hello", func(ctx context.Context, input *HelloRequest) (*HelloResponse, error) {
	    return &HelloResponse{Message: "Hello " + input.Name}, nil
	})

	rt, err := runtime.New(ctx,
	    runtime.WithFile("./plugin.wasm"),
	    runtime.WithHostFns(hostFn),
	)

# Type-Safe Function Calls

For type safety, you can create strongly-typed plugin function wrappers:

	type EchoRequest struct {
	    Message string
	}

	type EchoResponse struct {
	    Message string
	}

	fn, err := runtime.PluginFn[*EchoRequest, *EchoResponse](rt, "echo")
	if err != nil {
	    log.Fatalf("Failed to create function: %v", err)
	}

	resp, err := fn.Call(&EchoRequest{Message: "Hello"})
	if err != nil {
	    log.Fatalf("Failed to call function: %v", err)
	}

	fmt.Println(resp.Message)

# Raw Byte Functions

For functions that work directly with byte slices:

	byteFn, err := runtime.PluginFnByte(rt, "processBytes")
	if err != nil {
	    log.Fatalf("Failed to create function: %v", err)
	}

	result, err := byteFn.Call([]byte("raw data"))
	if err != nil {
	    log.Fatalf("Failed to call function: %v", err)
	}

	fmt.Printf("Result: %s\n", result)
*/
package hookr
