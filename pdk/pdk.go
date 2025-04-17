package pdk

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/tinylib/msgp/msgp"
)

type (
	// PluginFunction is a function that takes an input of type In and returns an output of type Out.
	// These are the concrete functions that are callable by the host.
	PluginFunction[In msgp.Unmarshaler, Out msgp.Marshaler] func(input In) (Out, error)

	// Function is a function that takes a byte slice as input and returns a byte slice as output.
	// This is the function that is called by the host.
	// It is a wrapper around the PluginFunction to allow for byte-based communication.
	Function func(input []byte) ([]byte, error)

	// Functions is a map of function names to their corresponding Function implementations.
	// This is the registry of functions that are callable by the host.
	// The key is the function name and the value is the Function implementation.
	Functions map[string]Function

	HostError struct {
		message string
	}
)

var allFns = Functions{}

func pluginFunction[In msgp.Unmarshaler, Out msgp.Marshaler](fn PluginFunction[In, Out]) Function {
	return func(input []byte) ([]byte, error) {
		var zero In
		t := reflect.TypeOf(zero)

		var pluginInput In

		// If it's a pointer type, create a new instance
		if t != nil && t.Kind() == reflect.Ptr {
			// Create a new instance of the element type
			elemType := t.Elem()
			newElem := reflect.New(elemType)
			pluginInput = newElem.Interface().(In)
		} else {
			// For non-pointer types, use the zero value
			pluginInput = zero
		}
		_, err := pluginInput.UnmarshalMsg(input) // unmarshal the input
		if err != nil {
			return nil, err
		}
		output, err := fn(pluginInput)
		if err != nil {
			return nil, err
		}

		return output.MarshalMsg(nil) // marshal the output
	}
}

// Fn adds a single function by name to the registry.
// This should be invoked in your initialize func to expose any functions you wish the host to use.
func Fn[In msgp.Unmarshaler, Out msgp.Marshaler](name string, fn PluginFunction[In, Out]) {
	allFns[name] = pluginFunction(fn)
}

// FnByte adds a single function by name to the registry.
// This should be invoked in your initialize func to expose any functions you wish the host to use.
func FnByte(name string, fn Function) {
	allFns[name] = fn
}

//go:export __plugin_call
func pluginCall(operationSize uint32, payloadSize uint32) bool {
	operation := make([]byte, operationSize) // alloc
	payload := make([]byte, payloadSize)     // alloc
	pluginRequest(bytesToPointer(operation), bytesToPointer(payload))

	if f, ok := allFns[string(operation)]; ok {
		response, err := f(payload)
		if err != nil {
			message := err.Error()
			pluginError(stringToPointer(message), uint32(len(message)))

			return false
		}

		pluginResponse(bytesToPointer(response), uint32(len(response)))

		return true
	}

	message := `Could not find function "` + string(operation) + `"`
	pluginError(stringToPointer(message), uint32(len(message)))

	return false
}

// Log is a convenience function to log messages to the console.
// It is a wrapper around the `__log` function.
// The message is passed as a string pointer and length to the host.
// If the message is empty, it will not log anything.
func Log(message string) {
	if len(message) == 0 {
		return
	}
	consoleLog(stringToPointer(message), uint32(len(message)))
}

type HostFunction[In msgp.Marshaler, Out msgp.Unmarshaler] struct {
	name string
}

func (h *HostFunction[In, Out]) Call(input In) (Out, error) {
	return Call[In, Out](h.name, input)
}

func HostFn[In msgp.Marshaler, Out msgp.Unmarshaler](name string) *HostFunction[In, Out] {
	return &HostFunction[In, Out]{name: name}
}

func Call[In msgp.Marshaler, Out msgp.Unmarshaler](operation string, input In) (Out, error) {
	var zero Out

	if reflect.ValueOf(input).Kind() == reflect.Ptr && reflect.ValueOf(input).IsNil() {
		return zero, fmt.Errorf("input cannot be nil")
	}

	data, err := input.MarshalMsg(nil)
	if err != nil {
		return zero, err
	}

	response, err := HostCall(operation, data)
	if err != nil {
		return zero, err
	}

	t := reflect.TypeOf(zero)

	var output Out

	// If it's a pointer type, create a new instance
	if t != nil && t.Kind() == reflect.Ptr {
		// Create a new instance of the element type
		elemType := t.Elem()
		newElem := reflect.New(elemType)
		output = newElem.Interface().(Out)
	} else {
		// For non-pointer types, use the zero value
		output = zero
	}

	output.UnmarshalMsg(response)
	if err != nil {
		return zero, err
	}
	return output, nil
}

type HostFunctionByte struct {
	name string
}

func (h *HostFunctionByte) Call(input []byte) ([]byte, error) {
	response, err := HostCall(h.name, input)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func HostFnByte(name string) *HostFunctionByte {
	return &HostFunctionByte{name: name}
}

// HostCall invokes an operation on the host.  The host uses `operation`
// to route to the `payload` to the appropriate operation.  The host will return
// a response payload if successful.
func HostCall(operation string, payload []byte) ([]byte, error) {
	result := hostCall(
		stringToPointer(operation), uint32(len(operation)),
		bytesToPointer(payload), uint32(len(payload)),
	)
	if !result {
		errorLen := hostErrorLen()
		message := make([]byte, errorLen) // alloc
		hostError(bytesToPointer(message))

		return nil, &HostError{message: string(message)} // alloc
	}

	responseLen := hostResponseLen()
	response := make([]byte, responseLen) // alloc
	hostResponse(bytesToPointer(response))

	return response, nil
}

//go:inline
func bytesToPointer(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

//go:inline
func stringToPointer(s string) uintptr {
	b := []byte(s)
	return bytesToPointer(b)
}

func (e *HostError) Error() string {
	return "Host error: " + e.message // alloc
}
