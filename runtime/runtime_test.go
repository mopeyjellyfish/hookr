package runtime

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"testing"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/mopeyjellyfish/hookr/runtime/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	SIMPLE_METHOD_WASM = "../testdata/simplemethod/bin/simplemethod.wasm"
	INVALID_WASM       = "../testdata/invalid/invalidformat.wasm"
	EMPTY_WASM         = "../testdata/empty/bin/empty.wasm"
)

var SIMPLE_METHOD_SCHEMA_HASH = [32]byte{
	'h', 'o', 'o', 'k', 'r', '-', 's', 'i', 'm', 'p', 'l', 'e', '-', 'm', 'e', 't',
	'h', 'o', 'd', '-', 's', 'c', 'h', 'e', 'm', 'a', '-', '0', '0', '0', '1', '!',
}

func HelloByte(ctx context.Context, input []byte) ([]byte, error) {
	name := string(input)
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	return []byte("Hello " + name), nil
}

func TestMethodHandler(t *testing.T) {
	e := Runtime{}
	data, err := e.methodHandler(context.Background(), 7, nil)
	require.Error(t, err)
	require.Nil(t, data)

	e.RegisterMethod(7, func(ctx context.Context, payload []byte) ([]byte, error) {
		return []byte("ok"), nil
	})

	data, err = e.methodHandler(context.Background(), 7, nil)
	require.NoError(t, err)
	require.Equal(t, []byte("ok"), data)
}

func TestHookrInvokeMethod(t *testing.T) {
	ctx := context.Background()
	hostHello := HostFnMethod(1, HelloByte)
	schema := runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}

	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(hostHello),
		WithContractSchema(schema),
	)
	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() {
		require.NoError(t, p.Close(ctx))
	}()

	resp, err := p.InvokeMethod(ctx, 2, []byte("Steve"))
	require.NoError(t, err)
	require.Equal(t, "Hello Steve", string(resp))
}

func TestHookrInvokeMethodHandshake(t *testing.T) {
	ctx := context.Background()
	schema := runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}

	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithContractSchema(schema),
	)
	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() {
		require.NoError(t, p.Close(ctx))
	}()

	resp, err := p.InvokeMethod(ctx, 2, []byte("Steve"))
	require.NoError(t, err)
	require.Equal(t, "Hello Steve", string(resp))
}

func TestHookrInvokeMethodHandshakeMismatch(t *testing.T) {
	ctx := context.Background()
	badHash := SIMPLE_METHOD_SCHEMA_HASH
	badHash[0] ^= 0xFF
	schema := runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: badHash,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}

	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithContractSchema(schema),
	)
	require.Error(t, err)
	require.Nil(t, p)
}

func TestHookrInvokeMethodCapabilitiesMismatch(t *testing.T) {
	ctx := context.Background()
	schema := runtimecontract.Schema{
		Name:         "simple-method",
		SchemaHash:   SIMPLE_METHOD_SCHEMA_HASH,
		Capabilities: 1 << 9,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}

	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithContractSchema(schema),
	)
	require.Error(t, err)
	require.Nil(t, p)
}

func TestHookrContractIntrospection(t *testing.T) {
	ctx := context.Background()
	schema := runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}

	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithContractSchema(schema),
	)
	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() {
		require.NoError(t, p.Close(ctx))
	}()

	require.True(t, p.SupportsMethodABI())
	require.False(t, p.HasPluginCapabilities(1<<5))
	require.Equal(t, uint64(0), p.PluginCapabilities())
	expected, ok := p.ExpectedHandshake()
	require.True(t, ok)
	require.Equal(t, SIMPLE_METHOD_SCHEMA_HASH, expected.SchemaHash)
	contractSchema, ok := p.ContractSchema()
	require.True(t, ok)
	require.Equal(t, "simple-method", contractSchema.Name)

	hs, ok := p.PluginHandshake()
	require.True(t, ok)
	require.Equal(t, SIMPLE_METHOD_SCHEMA_HASH, hs.SchemaHash)
}

func TestHookrWithContractSchema(t *testing.T) {
	ctx := context.Background()
	schema := runtimecontract.Schema{
		Name:         "simple-method",
		SchemaHash:   SIMPLE_METHOD_SCHEMA_HASH,
		Capabilities: 0,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}

	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithContractSchema(schema),
	)
	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() {
		require.NoError(t, p.Close(ctx))
	}()

	require.True(t, p.ContractHasMethodID(2))
	require.True(t, p.ContractHasMethodID(3))
	require.False(t, p.ContractHasMethodID(999))
	require.True(t, p.HasPluginMethodID(2))
	require.True(t, p.HasPluginMethodID(3))
	require.False(t, p.HasPluginMethodID(1))
	require.Equal(t, []uint32{2, 3}, p.PluginMethodIDs())
}

func TestHookrContractHandshakeNotSupported(t *testing.T) {
	ctx := context.Background()
	var hash [runtimecontract.SchemaHashLen]byte
	hash[0] = 1
	schema := runtimecontract.Schema{
		Name:       "empty",
		SchemaHash: hash,
		Methods: []runtimecontract.Method{
			{ID: 1, Name: "Ping", RequestType: "Empty", ResponseType: "Empty"},
		},
	}

	p, err := New(
		ctx,
		WithFile(EMPTY_WASM, WithAllowUnsigned()),
		WithContractSchema(schema),
	)
	require.Error(t, err)
	require.Nil(t, p)
}

func TestHookrHostError(t *testing.T) {
	ctx := context.Background()
	hostErr := func(context.Context, []byte) ([]byte, error) {
		return nil, errors.New("planned failure")
	}

	plugin, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, hostErr)),
	)
	require.NoError(t, err)
	require.NotNil(t, plugin)
	defer func() {
		require.NoError(t, plugin.Close(ctx))
	}()

	result, err := plugin.InvokeMethod(ctx, 2, []byte("Steve"))
	require.Error(t, err)
	require.Nil(t, result)
}

func TestHookrCompileTwice(t *testing.T) {
	ctx := context.Background()

	plugin, err := New(ctx, WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()))
	require.NoError(t, err)
	require.NotNil(t, plugin)
	defer func() {
		require.NoError(t, plugin.Close(ctx))
	}()

	err = plugin.Compile()
	require.Error(t, err)
}

func TestUninitializedHookr(t *testing.T) {
	e := Runtime{}
	require.Equal(t, uint32(0), e.MemorySize())
	require.Error(t, e.Compile())

	_, err := e.InvokeMethod(context.Background(), 2, nil)
	require.Error(t, err)

	require.Error(t, e.Init())
	require.Error(t, e.InitHookr())
	require.Error(t, e.InitRuntime())
	require.Error(t, e.Instantiate())
	require.NoError(t, e.Close(context.Background()))
}

func TestHookrInvalid(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile(INVALID_WASM, WithAllowUnsigned()))
	require.Error(t, err)
	require.Nil(t, plugin)
}

func TestHookrEmpty(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile(EMPTY_WASM, WithAllowUnsigned()))
	require.Error(t, err)
	assert.Nil(t, plugin)
}

func TestHookrOpts(t *testing.T) {
	plugin, err := New(context.Background(),
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithStderr(os.Stderr),
		WithStdout(os.Stdout),
		WithLogger(logger.Default),
		WithRandSource(rand.Reader),
	)
	require.NoError(t, err)
	require.NotNil(t, plugin)
	defer func() {
		require.NoError(t, plugin.Close(context.Background()))
	}()
}

func TestHookrBadHash(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile(SIMPLE_METHOD_WASM, WithHash("123")))
	require.Error(t, err)
	require.Nil(t, plugin)
}

func TestHookrUnknownFile(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile("unknown.wasm"))
	require.Error(t, err)
	require.Nil(t, plugin)
}

func TestHookrEmptyFile(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile(""))
	require.Error(t, err)
	require.Nil(t, plugin)
}

func TestHookrModule(t *testing.T) {
	ctx := context.Background()
	plugin, err := New(ctx, WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, plugin.Close(ctx))
	}()

	require.Equal(t, uint32(131072), plugin.MemorySize())
}
