package hookr

import (
	"context"
	"crypto/rand"
	"os"
	"testing"

	"github.com/mopeyjellyfish/hookr/host/logger"
	"github.com/mopeyjellyfish/hookr/testdata/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlugin(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "valid options",
			options: []Option{WithFile("./testdata/simple/bin/simple.wasm")},
			wantErr: false,
		},
		{
			name:    "invalid options",
			options: []Option{WithFile("")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPlugin(context.Background(), tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPlugin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Hello(ctx context.Context, input *api.HelloRequest) (*api.HelloResponse, error) {
	return &api.HelloResponse{
		Msg: "Hello " + input.Msg,
	}, nil
}

func TestPluginE2E(t *testing.T) {
	ctx := context.Background()
	hostFn := HostFn("hello", Hello)
	require.NotNil(t, hostFn, "failed to create host function")
	p, err := NewPlugin(ctx,
		WithFile("./testdata/simple/bin/simple.wasm"),
		WithHostFns(hostFn),
		WithLogger(logger.Default),
		WithStderr(os.Stderr),
		WithStdout(os.Stdout),
		WithRandSource(rand.Reader),
	)

	require.NoError(t, err, "failed to create module")
	require.NotNil(t, p, "plugin should not be nil")
	defer func() {
		err := p.Close(ctx)
		require.NoError(t, err, "failed to close module")
	}()

	echoFn, err := PluginFn[*api.EchoRequest, *api.EchoResponse](p, "echo")
	assert.NoError(t, err, "failed to create plugin function")
	vowelFn, err := PluginFnByte(p, "vowel")

	payload := &api.EchoRequest{
		Data: "Who controls the past controls the future; who controls the present controls the past.",
	}
	assert.NoError(t, err, "failed to create plugin function")
	assert.NotNil(t, echoFn, "plugin function should not be nil")
	d, err := echoFn.Call(payload) // confirm the call works
	assert.NotNil(t, d, "plugin function should return a value")
	assert.NoError(t, err, "failed to call plugin function")

	res, err := vowelFn.Call([]byte("aeiou"))
	assert.NoError(t, err, "failed to call plugin function")
	assert.NotNil(t, res, "plugin function should return a value")
	assert.Equal(t, []byte("5"), res, "plugin function should return 5 vowels")
}
