# Hookr

<p align="center">
    <strong>Seamless WebAssembly plugins for Go — secure, type-safe, and blazingly fast.</strong>
</p>

<p align="center">
    Extend your Go applications with dynamically loaded WASM modules while maintaining the performance and safety you expect.
</p>

---

<p align="center">
  <a href="https://github.com/mopeyjellyfish/hookr/actions/workflows/tests.yml"><img src="https://github.com/mopeyjellyfish/hookr/actions/workflows/tests.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/mopeyjellyfish/hookr/actions/workflows/lint.yml"><img src="https://github.com/mopeyjellyfish/hookr/actions/workflows/lint.yml/badge.svg" alt="Lint"></a>
  <img alt="GitHub Release" src="https://img.shields.io/github/v/release/mopeyjellyfish/hookr">
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
- [Performance Considerations](#performance-considerations)
- [Troubleshooting](#troubleshooting)
- [Project Structure](#project-structure)

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

Hookr is opinionated on how serialization occurs, it makes use of msgpack between host & plugin, and expects all requests and responses in Go to use github.com/tinylib/msgp which will generate high performant Marshal/Unmarshal to satisfy the interfaces used.

It is also possible to just use `[]byte` between host/plugin for the request and response data, and leave serialization/deserialization up to the caller.

### Type Generation

For JSON serialization, you can define standard Go structs with json tags:

```go
type MyRequest struct {
    Name    string `json:"name"`
    Count   int    `json:"count"`
    Enabled bool   `json:"enabled"`
}

type MyResponse struct {
    Result  string   `json:"result"`
    Items   []string `json:"items"`
    Success bool     `json:"success"`
}
```

For msgpack serialization, add the msgp tag and use code generation:

```go
//go:generate msgp

type MyRequest struct {
    Name    string `msg:"name"`
    Count   int    `msg:"count"`
    Enabled bool   `msg:"enabled"`
}

type MyResponse struct {
    Result  string   `msg:"result"`
    Items   []string `msg:"items"`
    Success bool     `msg:"success"`
}
```

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

2. **Generate Serialization Code**: Use the `msgp` tool to generate serialization code

```bash
# First, install the msgp generator tool
go install github.com/tinylib/msgp@latest

# Then generate the code (run from the root of your project)
go generate ./api/...
```

3. **Write Plugin Functions**: Implement functions that process the requests

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

4. **Register Functions**: Export your functions in the initialization function

```go
//go:wasmexport hookr_init
func Initialize() {
    pdk.RegisterFunction("hello", Hello)
    // Register more functions as needed
}
```

5. **Build the Plugin**: Compile your plugin using TinyGo

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

```
yourproject/
├── api/
│   ├── types.go           # Request/response type definitions
│   └── types_msgp.go      # Generated serialization code
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
