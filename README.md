# Hookr

Hookr is a Go library for securely loading and executing WebAssembly (WASM) plugins. It provides a simple, type-safe interface for communication between the host application and WASM plugins.

## Features

- **Type-safe plugin interface**: Strongly typed function calls between the host and plugins
- **Security**: Integrity verification of WASM modules through cryptographic hashing
- **Bi-directional communication**: Plugins can call back into the host
- **TinyGo compatibility**: Optimized for TinyGo WASM modules
- **Comprehensive logging**: Debug and trace plugin execution

## Installation

```bash
go get github.com/mopeyjellyfish/hookr
```

## Quick Start

### Host Application

Define some types for your WASM API:

```go
package api
//go:generate msgp
// Define request/response types
type EchoRequest struct {
    Message string `msg:"message"`
}

type EchoResponse struct {
    Message string `msg:"message"`
}
```

Use these types in your host application:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/mopeyjellyfish/hookr/host"
    "github.com/mopeyjellyfish/hookr/testdata/api"
)

func main() {
    // Create a new engine with the plugin
    ctx := context.Background()
    engine, err := host.New(ctx, host.WithFile("./plugin.wasm"))
    if err != nil {
        log.Fatalf("Failed to create engine: %v", err)
    }
    defer engine.Close(ctx)

    // Create type-safe callable
    echoFn, err := host.PluginFn[*api.EchoRequest, *api.EchoResponse](engine, "echo")
    if err != nil {
        log.Fatalf("Failed to create function: %v", err)
    }

    // Call the function
    resp, err := echoFn.Call(&api.EchoRequest{Message: "Hello from host!"})
    if err != nil {
        log.Fatalf("Function call failed: %v", err)
    }

    fmt.Printf("Plugin responded: %s\n", resp.Message)
}
```

### Plugin Code

```go
package main

import (
 "github.com/mopeyjellyfish/hookr/pdk"
 "github.com/mopeyjellyfish/hookr/testdata/api" // Use our API models
)

//go:wasmexport hookr_init
func Initialize() {
 // Register the echo function
 pdk.RegisterFunction("echo", Echo)
}

// Echo implements a simple echo service
func Echo(input *EchoRequest) (*EchoResponse, error) {
 // Log received message
 pdk.Log("Received message: " + input.Message)

 // Return the same message back
 return &EchoResponse{
  Message: input.Message,
 }, nil
}
```

### Building the Plugin

Use TinyGo to compile the plugin:

```bash
tinygo build -o bin/plugin.wasm -scheduler=none --no-debug -target=wasip1 -buildmode=c-shared main.go
```

## Advanced Usage

### Host Functions

Register host functions that can be called by the plugin, in the host application:

```go
// Define a host function, use a shared API package between host and plugin.
hostFn := func(input *api.GreetRequest) (*api.GreetResponse, error) {
 return &GreetResponse{
  Message: fmt.Sprintf("Hello, %s from host!", input.Name),
 }, nil
}

// Register the host function
greetFn := host.HostFn("greet", hostFn)

// Create engine with the host function
engine, err := host.New(ctx, 
    host.WithFile("./plugin.wasm"),
    host.WithHostFns(greetFn),
)
```

In the plugin, call the host function:

```go

//go:wasmexport hookr_init
func Initialize() {
 // Register the hello function for calling
 pdk.Fn("hello", Hello)
}

var Greet = pdk.HostFn[*GreetRequest, *GreetResponse]("greet")

func Hello(input *HelloRequest) (*HelloResponse, error) {
 // Call the host
 resp, err := Greet.Call(&GreetRequest{
  Name: input.Name,
 })
 if err != nil {
  return nil, err
 }
 
 return &HelloResponse{
  Message: resp.Message,
 }, nil
}
```

### Verifying Plugin Integrity

WASM plugins can be hash verified before loading:

```go
engine, err := host.New(ctx,
    host.WithFile("./plugin.wasm",
        host.WithHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
        host.WithHasher(host.Sha256Hasher{}),
    ),
)
```

## Project Structure

- `host/`: Used in host applications for loading and executing WASM modules.
- `pdk/`: Plugin Development Kit for building WASM plugins.

