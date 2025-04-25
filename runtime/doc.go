/*
Package runtime provides core functionality for loading, verifying, and executing WebAssembly modules.

The runtime package is the main interface for working with WASM plugins in Hookr. It handles
loading modules from files, verifying their integrity, and facilitating communication
between the runtime application and the plugins.

# Creating a new Runtime

A Runtime is the central component that manages a WASM plugin:

	ctx := context.Background()
	rt, err := runtime.New(ctx, runtime.WithFile("./plugin.wasm"))
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}
	defer rt.Close(ctx)

# Runtime Configuration

The Runtime can be configured with various options:

	rt, err := runtime.New(ctx,
		// Load plugin from file
		runtime.WithFile("./plugin.wasm"),

		// Configure I/O
		runtime.WithStdout(os.Stdout),
		runtime.WithStderr(os.Stderr),

		// Set a custom logger
		runtime.WithLogger(func(msg string) {
			log.Printf("[PLUGIN] %s", msg)
		}),

		// Register host functions
		runtime.WithHostFns(myHostFn),

		// Set a custom random source for deterministic behavior
		runtime.WithRandSource(myRandSource),
	)

# Invoking Plugin Functions

Plugin functions can be invoked directly with byte slices:

	result, err := rt.Invoke(ctx, "function_name", []byte("input data"))
	if err != nil {
		log.Fatalf("Function call failed: %v", err)
	}
	fmt.Printf("Result: %s\n", result)

# Type-Safe Function Calls

For type safety, you can create strongly-typed function wrappers:

	// Define request/response types that implement msgp.Marshaler/Unmarshaler
	type Request struct {
		Input string `msg:"input"`
	}

	type Response struct {
		Output string `msg:"output"`
	}

	// Create a type-safe function
	fn, err := runtime.PluginFn[*Request, *Response](engine, "process")
	if err != nil {
		log.Fatalf("Failed to create function wrapper: %v", err)
	}

	// Call the function
	resp, err := fn.Call(&Request{Input: "test data"})
	if err != nil {
		log.Fatalf("Function call failed: %v", err)
	}
	fmt.Printf("Output: %s\n", resp.Output)

# Registering Host Functions

Host functions allow the plugin to call back into the host application:

	// Define a runtime function
	helloFn := func(input *HelloRequest) (*HelloResponse, error) {
		return &HelloResponse{
			Message: fmt.Sprintf("Hello, %s!", input.Name),
		}, nil
	}

	// Register the runtime function
	hostFn := runtime.HostFn("hello", helloFn)

	engine, err := runtime.New(ctx,
		runtime.WithFile("./plugin.wasm"),
		runtime.WithHostFns(hostFn),
	)

# File Integrity

To ensure the integrity of WASM files, you can use hashing:

	engine, err := runtime.New(ctx,
		runtime.WithFile("./plugin.wasm",
			runtime.WithHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
			runtime.WithHasher(runtime.Sha256Hasher{}),
		),
	)

# Memory Management

You can query memory usage of the WASM module:

	memSize := rt.MemorySize()
	fmt.Printf("WASM module memory size: %d bytes\n", memSize)
*/
package runtime
