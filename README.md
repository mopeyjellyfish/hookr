# Hookr

<p align="center">
    <strong>Seamless WebAssembly plugins for Go — secure, type-safe, and blazingly fast.</strong>
</p>

<p align="center">
    Extend Go applications with dynamically loaded WASM modules.
</p>

---

<p align="center">
  <a href="https://github.com/mopeyjellyfish/hookr/actions/workflows/tests.yml"><img src="https://github.com/mopeyjellyfish/hookr/actions/workflows/tests.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/mopeyjellyfish/hookr/actions/workflows/lint.yml"><img src="https://github.com/mopeyjellyfish/hookr/actions/workflows/lint.yml/badge.svg" alt="Lint"></a>
  <img alt="GitHub Release" src="https://img.shields.io/github/v/release/mopeyjellyfish/hookr">
  <a href="https://codecov.io/github/mopeyjellyfish/hookr" > 
     <img src="https://codecov.io/github/mopeyjellyfish/hookr/graph/badge.svg?token=peUgWB4joM"/> 
  </a>
</p>

## Features

- **Type-safe plugin interface**: Strongly typed function calls between the host and plugins
- **Security**: Integrity verification of WASM modules through cryptographic hashing
- **Bi-directional communication**: Plugins can call back into the host
- **TinyGo compatibility**: Optimized for TinyGo WASM modules
- **Comprehensive logging**: Debug and trace plugin execution
- **Simple API**: All functionality exposed through a single import path

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [API Overview](#api-overview)
- [Plugin Development](#plugin-development)
- [Advanced Usage](#advanced-usage)
- [Project Structure](#project-structure)
- [PDK](## PDK Support)

## Installation

```bash
go get github.com/mopeyjellyfish/hookr
```

### Prerequisites

- Go 1.24 or higher
- TinyGo 0.30.0 or higher (for building plugins)

## Quick Start

### Host Application

```go
package main

import (
 "context"
 "fmt"
 "log"

 "github.com/mopeyjellyfish/hookr"
)

// Define request/response types
type EchoRequest struct {
 Message string `json:"message"`
}

type EchoResponse struct {
 Message string `json:"message"`
}

func main() {
 // Create a new plugin
 ctx := context.Background()
 plugin, err := hookr.NewPlugin(ctx, hookr.WithFile("./plugin.wasm"))
 if err != nil {
  log.Fatalf("Failed to create plugin: %v", err)
 }
 defer plugin.Close(ctx)

 // Create type-safe function wrapper
 echoFn, err := hookr.PluginFn[*EchoRequest, *EchoResponse](plugin, "echo")
 if err != nil {
  log.Fatalf("Failed to create function: %v", err)
 }

 // Call the function
 resp, err := echoFn.Call(&EchoRequest{Message: "Hello from host!"})
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
)

// Define request/response types
type EchoRequest struct {
 Message string `json:"message"`
}

type EchoResponse struct {
 Message string `json:"message"`
}

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
tinygo build -o plugin.wasm -scheduler=none --no-debug -target=wasip1 -buildmode=c-shared main.go
```

## API Overview

Hookr provides a streamlined API for communication between the host application and WASM plugins.

### Core Components

- **Plugin**: Interface representing a loaded WASM plugin
- **HostFn**: For registering callback functions that plugins can call
- **PluginFn**: For creating type-safe wrappers around plugin functions

### Data Exchange

Hosts send data to WASM plugins, and plugins send data back to the host. Hookr uses serialization to convert Go types to a format that can be safely passed across the WebAssembly boundary.

### Serialization Options

Anything which implements the Marshal or Unmarshal interfaces can be used to serialize and deserialize in both WASM and the host application. Currently all the examples are for MessagePack, see the following section.

#### MessagePack

Hookr primarily uses [MessagePack](https://github.com/tinylib/msgp) for efficient binary serialization between host and plugins. The framework provides type-safe functions with generics support:

```go
// In the host application:
// Create a strongly-typed function wrapper
msgpFn, err := hookr.PluginFnSerial[*api.Request, *api.Response](plugin, "function_name")

// Register a host function for plugin callbacks
hostFn := hookr.HostFnSerial("operation_name", MyHostFunction)
```

```go
// In your plugin:
// Register a function to handle host calls
pdk.Fn("function_name", MyPluginFunction)

// Create a wrapper to call host functions
var hostOp = pdk.HostFn[*api.Request, *api.Response]("operation_name")
```

For this to work, your types must implement `Marshaler` and `Unmarshaler` interfaces, typically generated with:

```go
//go:generate msgp
type Request struct {
    Input string `msg:"input"`
}
```

#### Raw Bytes

For custom serialization needs or direct binary data handling, Hookr provides byte-based functions:

```go
// In the host
byteFn, err := hookr.PluginFnByte(plugin, "raw_operation")
result, err := byteFn.Call([]byte("raw data"))

// Register a byte-based host function
byteFn := hookr.HostFnByte("byte_operation", func(ctx context.Context, data []byte) ([]byte, error) {
    // Process raw bytes
    return processedData, nil
})
```

```go
// In the plugin
pdk.FnByte("raw_operation", func(data []byte) ([]byte, error) {
    // Process incoming byte slice
    return processedBytes, nil
})

// Call a byte-based host function
var hostByteOp = pdk.HostFnByte("byte_operation")
result, err := hostByteOp.Call(myData)
```

This approach gives you complete control over serialization while still leveraging Hookr's WebAssembly integration.

## Plugin Development

Developing plugins for Hookr involves the following steps:

1. **Define Types**: Create Go structs for the request and response types

```go
// api/types.go
package api

//go:generate msgp

// HelloRequest is sent to the plugin
type HelloRequest struct {
    Name string `msg:"name" json:"name"`
}

// HelloResponse is returned from the plugin
type HelloResponse struct {
    Message string `msg:"message" json:"message"`
}
```

1. **Generate Serialization Code**: Use the `msgp` tool to generate serialization code

```bash
# First, install the msgp generator tool
go install github.com/tinylib/msgp@latest

# Then generate the code (run from the root of your project)
go generate ./api/...
```

1. **Write Plugin Functions**: Implement functions that process the requests

```go
package main

import (
    "fmt"
    
    "github.com/mopeyjellyfish/hookr/pdk"
    "yourproject/api"
)

// Hello function handles HelloRequest and returns HelloResponse
func Hello(input *api.HelloRequest) (*api.HelloResponse, error) {
    pdk.Log(fmt.Sprintf("Hello called with name: %s", input.Name))
    
    return &api.HelloResponse{
        Message: fmt.Sprintf("Hello, %s!", input.Name),
    }, nil
}
```

1. **Register Functions**: Export your functions in the initialization function

```go
//go:wasmexport hookr_init
func Initialize() {
    pdk.RegisterFunction("hello", Hello)
    // Register more functions as needed
}
```

1. **Build the Plugin**: Compile your plugin using TinyGo

```bash
tinygo build -o plugin.wasm -scheduler=none --no-debug -target=wasip1 -buildmode=c-shared main.go
```

### Host-Plugin Communication Pattern

Hookr uses a well-defined pattern for communication:

1. **Host sends request**: The host serializes a request struct and sends it to the plugin
2. **Plugin processes request**: The plugin deserializes the request, processes it, and serializes a response
3. **Plugin returns response**: The response is sent back to the host and deserialized

### Shared API Package

For best results, create a shared API package that defines all the request and response types used by both the host and plugins:

```sh
yourproject/
├── api/
│   ├── types.go           # Request/response type definitions
│   └── types_gen.go       # Generated serialization code
├── host/
│   └── main.go            # Host application
└── plugin/
    └── main.go            # Plugin implementation
```

This approach ensures type consistency between the host and plugins.

## Advanced Usage

### Host Functions

Register host functions that can be called by the plugin, in the host application:

```go
// Define a host function, use a shared API package between host and plugin.
func Greet(ctx context.Context, input *api.GreetRequest) (*api.GreetResponse, error) {
 return &api.GreetResponse{
  Message: fmt.Sprintf("Hello, %s from host!", input.Name),
 }, nil
}

// Create a host function
greetFn := hookr.HostFn("greet", Greet)

// Create plugin with the host function
plugin, err := hookr.NewPlugin(ctx, 
    hookr.WithFile("./plugin.wasm"),
    hookr.WithHostFns(greetFn),
)
```

In the plugin, call the host function:

```go
//go:wasmexport hookr_init
func Initialize() {
 // Register the hello function for calling from the host
 pdk.Fn("hello", Hello)
}

// Create a type-safe function wrapper for the host function
var Greet = pdk.HostFn[*api.GreetRequest, *api.GreetResponse]("greet")

func Hello(input *api.HelloRequest) (*api.HelloResponse, error) {
 // Call the host
 resp, err := Greet.Call(&api.GreetRequest{
  Name: input.Name,
 })
 if err != nil {
  return nil, err
 }
 
 return &api.HelloResponse{
  Message: resp.Message,
 }, nil
}
```

### Raw Byte Functions

For functions that don't need structured data or when you want to handle serialization yourself:

```go
// In the host application
byteFn, err := hookr.PluginFnByte(plugin, "countVowels")
if err != nil {
  log.Fatalf("Failed to create function: %v", err)
}

result, err := byteFn.Call([]byte("hello world"))
if err != nil {
  log.Fatalf("Failed to call function: %v", err)
}

fmt.Printf("Vowel count: %s\n", result)
```

```go
// In the plugin
//go:wasmexport hookr_init
func Initialize() {
  // Register a byte function
  pdk.FnByte("countVowels", CountVowels)
}

func CountVowels(input []byte) ([]byte, error) {
  text := string(input)
  count := 0
  for _, c := range text {
    switch c {
    case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
      count++
    }
  }
  return []byte(fmt.Sprintf("%d", count)), nil
}
```

### Verifying Plugin Integrity

WASM plugins can be hash verified before loading:

```go
plugin, err := hookr.NewPlugin(ctx,
    hookr.WithFile("./plugin.wasm",
        hookr.WithHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
        hookr.WithHasher(hookr.Sha256Hasher{}),
    ),
)
```

## Project Structure

- `hookr/`: Main package for host applications loading and executing WASM plugins
- `hookr/pdk/`: Plugin Development Kit for building WASM plugins in Go

## PDK Support

Hookr currently supports the following languages for plugin development:

| Language       | Support Level | Notes                                    |
|----------------|---------------|------------------------------------------|
| Go             | Full          | Using TinyGo compiler for WASM modules   |
| Rust           | Planned       | Coming in future releases                |
| Zig            | Planned       | Coming in future releases                |
| AssemblyScript | Planned       | Coming in future releases                |
| C/C++          | Planned       | Coming in future releases                |
