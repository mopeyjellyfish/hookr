package main

// #include <stdlib.h>
import "C"

import (
	"fmt"

	"github.com/mopeyjellyfish/hookr/examples/simple/host"
	"github.com/mopeyjellyfish/hookr/pkg/pdk/memory"
)

//export blank
func blank() uint64 {
	return 0
}

//export returnError
func returnError() uint64 {
	return 1
}

//export echo
func echo(ptr uint64) uint64 {
	output := fmt.Sprintf("echo: %s", memory.ReadString(ptr))
	return memory.WriteString(output)
}

//export echojson
func echojson(ptr uint64) uint64 {
	input := memory.ReadString(ptr)
	output := map[string]interface{}{"echo": input}
	return memory.WriteJson(output)
}

//export echoRandNumber
func echoRandNumber() uint64 {
	output := host.RandomNumber(0, 100000, 10)
	echoOutput := map[string]interface{}{"randomEcho": output.Numbers}
	return memory.WriteJson(echoOutput)
}

//export hostEchoString
func hostEchoString(ptr uint64) uint64 {
	data := memory.ReadString(ptr)
	hostData := host.HostPrintString(data)
	return memory.WriteString(hostData)
}

//export getRandomString
func getRandomString() uint64 {
	output := host.RandomString(10)
	return memory.WriteJson(output)
}

//export nothing
func nothing() {
}

//export panicFn
func panicFn() {
	panic("panic")
}

//export badPtr
func badPtr() uint64 {
	return 3
}

// main is required for the `wasi` target, even if it isn't used.
// See https://wazero.io/languages/tinygo/#why-do-i-have-to-define-main
func main() {}
