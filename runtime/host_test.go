package runtime

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"testing"

	"github.com/mopeyjellyfish/hookr/runtime/logger"
	"github.com/mopeyjellyfish/hookr/testdata/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	SIMPLE_WASM  = "../testdata/simple/bin/simple.wasm"
	INVALID_WASM = "../testdata/invalid/invalidformat.wasm"
	EMPTY_WASM   = "../testdata/empty/bin/empty.wasm"
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

func HelloError(ctx context.Context, input *api.HelloRequest) (*api.HelloResponse, error) {
	return nil, errors.New("planned failure")
}

func TestHookr(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		file string
	}{
		{"simple", SIMPLE_WASM},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, err := New(ctx, WithFile(test.file), WithHostFns(HostFnMsgp("hello", Hello)))
			require.NoError(t, err, "failed to create module")
			require.NotNil(t, p, "plugin should not be nil")
			defer func() {
				err := p.Close(ctx)
				require.NoError(t, err, "failed to close module")
			}()
			fn, err := PluginFnMsgp[*api.EchoRequest, *api.EchoResponse](p, "echo")
			require.NoError(t, err, "failed to create plugin function")
			require.NotNil(t, fn, "plugin function should not be nil")
			resp, err := fn.Call(context.Background(), &api.EchoRequest{
				Data: "Steve",
			})
			require.NoError(t, err, "failed to invoke echo")
			require.Equal(
				t,
				"Hello Steve",
				resp.Data,
				"echo did not return the expected payload",
			)
		})
	}
}

func TestHookrByte(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		file string
	}{
		{"simple", SIMPLE_WASM},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, err := New(ctx, WithFile(test.file), WithHostFns(HostFnByte("helloByte", HelloByte)))
			require.NoError(t, err, "failed to create module")
			require.NotNil(t, p, "plugin should not be nil")
			defer func() {
				err := p.Close(ctx)
				require.NoError(t, err, "failed to close module")
			}()

			fn, err := PluginFnByte(p, "echoByte")
			require.NoError(t, err, "failed to create plugin function")
			resp, err := fn.Call(context.Background(), []byte("Steve"))
			require.NoError(t, err, "failed to invoke echo")
			require.Equal(
				t,
				"Hello Steve",
				string(resp),
				"echo did not return the expected payload",
			)
		})
	}
}

func TestHookrHostFnError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		file string
	}{
		{"simple", SIMPLE_WASM},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, err := New(ctx, WithFile(test.file), WithHostFns(HostFnMsgp("hello", HelloError)))
			require.NoError(t, err, "failed to create module")
			require.NotNil(t, p, "plugin should not be nil")
			defer func() {
				err := p.Close(ctx)
				require.NoError(t, err, "failed to close module")
			}()

			fn, err := PluginFnMsgp[*api.EchoRequest, *api.EchoResponse](p, "echo")
			require.NoError(t, err, "failed to create plugin function")
			resp, err := fn.Call(context.Background(), &api.EchoRequest{
				Data: "Steve",
			})
			require.Error(t, err, "expected error from invoking echo due to host error")
			require.Nil(t, resp, "result should be nil")
		})
	}
}

func TestHookrCompileTwice(t *testing.T) {
	ctx := context.Background()

	plugin, err := New(ctx, WithFile(SIMPLE_WASM))
	require.NoError(t, err, "failed to create module")
	require.NotNil(t, plugin, "plugin should not be nil")
	defer func() {
		err := plugin.Close(ctx)
		require.NoError(t, err, "failed to close module")
	}()

	err = plugin.Compile()
	require.Error(t, err, "expected error when compiling nil module")
}

func TestUninitializedHookr(t *testing.T) {
	e := Runtime{}
	size := e.MemorySize()
	require.Equal(t, uint32(0), size, "Memory size should be 0 bytes")

	err := e.Compile()
	require.Error(t, err, "expected error when compiling nil module")

	_, err = e.Invoke(context.Background(), "echo", nil)
	require.Error(t, err, "expected error when invoking on uninitialized engine")

	err = e.Init()
	require.Error(t, err, "expected error when initializing uninitialized engine")

	err = e.InitHookr()
	require.Error(t, err, "expected error when initializing hookr on uninitialized engine")

	err = e.InitRuntime()
	require.Error(t, err, "expected error when initializing runtime on uninitialized engine")

	err = e.Instantiate()
	require.Error(t, err, "expected error when instantiating uninitialized engine")

	err = e.Close(context.Background())
	require.NoError(t, err, "expected no error when closing uninitialized engine")
}

func TestHookrInvalid(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		file string
	}{
		{"invalid", INVALID_WASM},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plugin, err := New(ctx, WithFile(INVALID_WASM))
			require.Error(t, err, "expected error when loading invalid wasm")
			require.Nil(t, plugin, "plugin should be nil")
		})
	}
}

func TestHookrError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		file string
	}{
		{"simple", SIMPLE_WASM},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plugin, err := New(ctx, WithFile(test.file))
			require.NoError(t, err, "failed to create module")
			require.NotNil(t, plugin, "plugin should not be nil")
			defer func() {
				err := plugin.Close(ctx)
				require.NoError(t, err, "failed to close module")
			}()
			payload := []byte("Hello, World!")
			result, err := plugin.Invoke(ctx, "nope", payload)
			require.Error(t, err, "expected error from invoking nope")
			require.Nil(t, result, "nope should return nil")
		})
	}
}

func TestHookrHostError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		file string
	}{
		{"simple", SIMPLE_WASM},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hostErr := func(context.Context, string, []byte) ([]byte, error) {
				return nil, errors.New("Planned Failure")
			}
			plugin, err := New(ctx, WithFile(test.file), WithCallHandler(hostErr))
			require.NoError(t, err, "failed to create module")
			require.NotNil(t, plugin, "plugin should not be nil")
			defer func() {
				err := plugin.Close(ctx)
				require.NoError(t, err, "failed to close module")
			}()
			payload := []byte("Hello, World!")
			result, err := plugin.Invoke(ctx, "echo", payload)
			require.Error(t, err, "expected error from invoking echo due to host error")
			require.Nil(t, result, "echo should return nil")
		})
	}
}

func TestHookrEmpty(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		file string
	}{
		{"empty", EMPTY_WASM},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plugin, err := New(ctx, WithFile(test.file))
			require.Error(t, err, "failed to create module")
			assert.Nil(t, plugin, "plugin should be nil on error")
		})
	}
}

func TestHookrOpts(t *testing.T) {
	callHandler := func(ctx context.Context, operation string, payload []byte) ([]byte, error) {
		return nil, nil
	}
	plugin, err := New(context.Background(),
		WithFile(SIMPLE_WASM, WithHasher(DefaultHasher{})),
		WithCallHandler(callHandler),
		WithStderr(os.Stderr),
		WithStdout(os.Stdout),
		WithLogger(logger.Default),
		WithRandSource(rand.Reader),
	)
	require.NoError(t, err, "failed to create module")
	require.NotNil(t, plugin, "plugin should not be nil")
}

func TestHookrBadHash(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile(SIMPLE_WASM, WithHash("123")))
	require.Error(t, err, "expected error when loading invalid hasher")
	require.Nil(t, plugin, "plugin should be nil")
}

func TestHookrUnknownFile(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile("unknown.wasm"))
	require.Error(t, err, "expected error when loading unknown file")
	require.Nil(t, plugin, "plugin should be nil")
}

func TestHookrEmptyFile(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile(""))
	require.Error(t, err, "expected error when loading empty file")
	require.Nil(t, plugin, "plugin should be nil")
}

func TestHookrModule(t *testing.T) {
	ctx := context.Background()
	t.Run("MemorySize", func(t *testing.T) {
		plugin, err := New(ctx, WithFile(SIMPLE_WASM))
		require.NoError(t, err, "failed to create module")
		defer func() {
			err := plugin.Close(ctx)
			require.NoError(t, err, "failed to close module")
		}()

		memorySize := plugin.MemorySize()
		require.Equal(t, uint32(131072), memorySize, "Memory size should be 65536 bytes")
	})
}

func TestPluginFn(t *testing.T) {
	ctx := context.Background()
	_, err := PluginFnMsgp[*api.EchoRequest, *api.EchoResponse](nil, "test")
	require.Error(t, err, "expected error when creating plugin function with nil engine")

	hostFn := HostFnMsgp("hello", Hello)
	p, err := New(ctx, WithFile(SIMPLE_WASM), WithHostFns(hostFn))
	require.NoError(t, err, "failed to create module")
	defer func() {
		err := p.Close(ctx)
		require.NoError(t, err, "failed to close module")
	}()
	_, err = PluginFnMsgp[*api.EchoRequest, *api.EchoResponse](p, "")
	require.Error(t, err, "expected error when creating plugin function with empty name")
}

func TestPluginFnCalls(t *testing.T) {
	ctx := context.Background()
	hostFn := HostFnMsgp("hello", Hello)
	p, err := New(ctx, WithFile(SIMPLE_WASM), WithHostFns(hostFn))
	require.NoError(t, err, "failed to create module")
	fn, err := PluginFnMsgp[*api.EchoRequest, *api.EchoResponse](p, "echo")
	require.NoError(t, err, "expected error when creating plugin function with empty name")

	resp, err := fn.Call(context.Background(), nil)
	require.Error(t, err, "expected error when calling plugin function with nil input")
	require.Nil(t, resp, "expected nil response when calling plugin function with nil input")
}
