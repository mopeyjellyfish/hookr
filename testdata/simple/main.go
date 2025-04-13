package main

import (
	"fmt"

	"github.com/mopeyjellyfish/hookr/pdk"
	"github.com/mopeyjellyfish/hookr/testdata/api"
)

//go:wasmexport hookr_init
func Initialize() {
	// Register echo and fail functions
	pdk.Fn("echo", Echo)
	pdk.Fn("nope", Fail)
}

var Hello = pdk.HostFn[*api.HelloRequest, *api.HelloResponse]("hello")

// Echo will return the payload
func Echo(payload *api.EchoRequest) (*api.EchoResponse, error) {
	// Callback with Payload
	resp, err := Hello.Call(&api.HelloRequest{
		Msg: string(payload.Data),
	})
	if err != nil {
		pdk.Log(err.Error())
		return nil, err
	}
	echoResp := &api.EchoResponse{
		Data: resp.Msg,
	}
	return echoResp, nil
}

// Fail will return an error when called
func Fail(payload *api.HelloRequest) (*api.EchoResponse, error) {
	return nil, fmt.Errorf("planned Failure")
}
