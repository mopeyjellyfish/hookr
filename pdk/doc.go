/*
Package pdk provides the Plugin Development Kit for creating WASM plugins compatible with Hookr.

The PDK package contains the necessary tools and utilities for developers to build
WASM plugins that can be loaded and executed by Hookr host applications. It handles
the communication between the plugin and the host, providing a simple and type-safe API.

# Creating a Plugin

To create a plugin, you need to register your functions and export the initialization function:

	package main

	import (
		"github.com/mopeyjellyfish/hookr/pdk"
	)

	//go:wasmexport hookr_init
	func Initialize() {
		// Register your functions
		pdk.Fn("hello", Hello)
		pdk.Fn("echo", Echo)
	}

	// Hello is a simple function that returns a greeting
	func Hello(input *HelloRequest) (*HelloResponse, error) {
		return &HelloResponse{
			Message: "Hello, " + input.Name + "!",
		}, nil
	}

	// Echo returns the input as output
	func Echo(input *EchoRequest) (*EchoResponse, error) {
		return &EchoResponse{
			Data: input.Data,
		}, nil
	}

# API

Host's are able to send data to a WASM plugin and a WASM plugin is able to send data back to the host.
We use the msgp package to serialize and deserialize data between the host and the plugin.

1. Write your plugin structs in standard Go types.
2. Use the `//go:generate msgp` directive to generate the serialization code.
3. Use the structs as input and output types for your plugin functions.
4. Register your functions with the `pdk.Fn` function.
5. Implement the plugin functions to handle the input and return the output.
6. On the host side, you will also make use of the same API package for the inputs and outputs for both host functions and also for calling plugin functions.

# Function Signatures

Plugin functions must follow this pattern:

	func YourFunction(input InputType) (OutputType, error)

Where InputType and OutputType implement the msgp.Unmarshaler and msgp.Marshaler interfaces
respectively. The easiest way is to use structs with msgp tags:

	//go:generate msgp

	type HelloRequest struct {
		Name string `msg:"name"`
	}

	type HelloResponse struct {
		Message string `msg:"message"`
	}

# Calling Host Functions

Plugins can call back into the host using the HostFn function:

	var GreetHost = pdk.HostFn[*GreetRequest, *GreetResponse]("greet")

	func Hello(input *HelloRequest) (*HelloResponse, error) {
		// Call the host
		resp, err := GreetHost.Call(&GreetRequest{
			Name: input.Name,
		})
		if err != nil {
			return nil, err
		}

		return &HelloResponse{
			Message: resp.Greeting,
		}, nil
	}

# Logging

The PDK provides a logging function that sends messages to the host:

	func Hello(input *HelloRequest) (*HelloResponse, error) {
		pdk.Log("Received hello request for: " + input.Name)

		// Function logic...

		return &HelloResponse{
			Message: "Hello, " + input.Name + "!",
		}, nil
	}

# Error Handling

Errors returned from plugin functions are properly propagated to the host:

	func Divide(input *DivideRequest) (*DivideResponse, error) {
		if input.Divisor == 0 {
			return nil, fmt.Errorf("division by zero")
		}

		return &DivideResponse{
			Result: input.Dividend / input.Divisor,
		}, nil
	}

# Building Plugins

To build a plugin for use with Hookr, you typically use TinyGo:

	tinygo build -o plugin.wasm -scheduler=none --no-debug -target=wasi main.go

This produces a WebAssembly module that can be loaded by a Hookr host application.
*/
package pdk
