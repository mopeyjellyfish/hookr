package hookr

import (
	"context"
	"errors"
	"testing"

	"github.com/mopeyjellyfish/hookr/runtime"
	"github.com/mopeyjellyfish/hookr/testdata/api"
	"github.com/stretchr/testify/require"
)

func Hello(ctx context.Context, input *api.HelloRequest) (*api.HelloResponse, error) {
	return &api.HelloResponse{
		Msg: "Hello " + input.Msg,
	}, nil
}

func HelloByte(ctx context.Context, input []byte) ([]byte, error) {
	name := string(input)
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	helloName := "Hello " + name
	helloNameBytes := []byte(helloName)
	return helloNameBytes, nil
}

func TestHookr(t *testing.T) {
	// setting up the plugin
	helloFn := runtime.HostFnMsgp("hello", Hello)
	hellBytesFn := runtime.HostFnByte("helloByte", HelloByte)
	require.NotNil(t, helloFn, "host function should not be nil")
	rt, err := runtime.New(
		context.Background(),
		runtime.WithFile("./testdata/simple/bin/simple.wasm"),
		runtime.WithHostFns(helloFn, hellBytesFn),
	)
	require.NoError(t, err, "failed to create module")
	require.NotNil(t, rt, "plugin should not be nil")

	// retrieving callable plugin functions
	byteFn, err := runtime.PluginFnByte(rt, "echoByte")
	require.NotNil(t, byteFn, "plugin function should not be nil")
	require.NoError(t, err, "failed to create plugin function")

	msgFn, err := runtime.PluginFnMsgp[*api.EchoRequest, *api.EchoResponse](rt, "echo")
	require.NotNil(t, msgFn, "plugin function should not be nil")
	require.NoError(t, err, "failed to create plugin function")

	// making plugin calls
	payload := &api.EchoRequest{
		Data: "Who controls the past controls the future; who controls the present controls the past.",
	}
	d, err := msgFn.Call(context.Background(), payload) // confirm the call works
	require.NoError(t, err, "failed to call plugin function")
	require.NotNil(t, d, "plugin function should return a value")

	payloadByte := []byte(
		"Who controls the past controls the future; who controls the present controls the past.",
	)
	dByte, err := byteFn.Call(context.Background(), payloadByte) // confirm the call works
	require.NoError(t, err, "failed to call plugin function")
	require.NotNil(t, dByte, "plugin function should return a value")
}
