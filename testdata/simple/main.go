package main

import (
	"fmt"

	hookr "github.com/mopeyjellyfish/hookr/pdk"
	"github.com/mopeyjellyfish/hookr/testdata/api"
)

//go:wasmexport hookr_init
func Initialize() {
	// Register echo and fail functions
	hookr.FnSerial("echo", Echo) // A function that the host imports from the plugin
	hookr.FnSerial("nope", Fail)

	hookr.FnByte("echoByte", EchoByte)
	hookr.FnByte("vowel", Vowels)
}

var Hello = hookr.HostFnSerial[*api.HelloRequest, *api.HelloResponse]("hello") // A function that the host exposes to the plugin
var HelloByte = hookr.HostFnByte("helloByte")

// Echo will return the payload
func Echo(payload *api.EchoRequest) (*api.EchoResponse, error) {
	// Callback with Payload
	resp, err := Hello.Call(&api.HelloRequest{
		Msg: string(payload.Data),
	})
	if err != nil {
		hookr.Log(err.Error())
		return nil, err
	}
	echoResp := &api.EchoResponse{
		Data: resp.Msg,
	}
	return echoResp, nil
}

func Vowels(payload []byte) ([]byte, error) {
	// Counts the number of vowels in the payload
	vowelCount := 0
	for _, b := range payload {
		if b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u' ||
			b == 'A' || b == 'E' || b == 'I' || b == 'O' || b == 'U' {
			vowelCount++
		}
	}
	// Convert the count to a byte slice
	vowelCountBytes := []byte(fmt.Sprintf("%d", vowelCount))
	return vowelCountBytes, nil
}

func EchoByte(payload []byte) ([]byte, error) {
	// Callback with Payload
	resp, err := HelloByte.Call(payload)
	if err != nil {
		hookr.Log(err.Error())
		return nil, err
	}

	return resp, nil
}

// Fail will return an error when called
func Fail(payload *api.HelloRequest) (*api.EchoResponse, error) {
	return nil, fmt.Errorf("planned Failure")
}
