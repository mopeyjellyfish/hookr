package runtime

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"os"
	"testing"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/mopeyjellyfish/hookr/runtime/invoke"
	"github.com/mopeyjellyfish/hookr/runtime/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	SIMPLE_METHOD_WASM          = "../testdata/simplemethod/bin/simplemethod.wasm"
	INVALID_WASM                = "../testdata/invalid/invalidformat.wasm"
	EMPTY_WASM                  = "../testdata/empty/bin/empty.wasm"
	HANDSHAKE_NOABI             = "../testdata/handshake_noabi/bin/handshake_noabi.wasm"
	HANDSHAKE_NOHANDSHAKE       = "../testdata/handshake_nohandshake/bin/handshake_nohandshake.wasm"
	HANDSHAKE_NOHASH            = "../testdata/handshake_nohash/bin/handshake_nohash.wasm"
	HANDSHAKE_BADHASHLEN        = "../testdata/handshake_badhashlen/bin/handshake_badhashlen.wasm"
	HANDSHAKE_BADHASHPTR        = "../testdata/handshake_badhashptr/bin/handshake_badhashptr.wasm"
	HANDSHAKE_CAPS_NORESULTS    = "../testdata/handshake_caps_noresults/bin/handshake_caps_noresults.wasm"
	HANDSHAKE_CAPS_TRAP         = "../testdata/handshake_caps_trap/bin/handshake_caps_trap.wasm"
	HANDSHAKE_NOMETHODS         = "../testdata/handshake_nomethods/bin/handshake_nomethods.wasm"
	HANDSHAKE_BADMETHODS        = "../testdata/handshake_badmethods/bin/handshake_badmethods.wasm"
	HANDSHAKE_EMPTYMETHODS      = "../testdata/handshake_emptymethods/bin/handshake_emptymethods.wasm"
	HANDSHAKE_METHODS_NORESULTS = "../testdata/handshake_methods_noresults/bin/handshake_methods_noresults.wasm"
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
	t.Cleanup(func() {
		require.NoError(t, plugin.Close(context.Background()))
	})
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

func TestWithContractSchemaRejectsInvalidSchema(t *testing.T) {
	rt := &Runtime{}
	err := WithContractSchema(runtimecontract.Schema{})(rt)
	require.Error(t, err)
}

func TestInitRuntimePropagatesError(t *testing.T) {
	rt := &Runtime{
		ctx: context.Background(),
		newRuntime: func(context.Context) (wazero.Runtime, error) {
			return nil, errors.New("planned runtime failure")
		},
	}
	err := rt.InitRuntime()
	require.EqualError(t, err, "planned runtime failure")
	require.Nil(t, rt.r)
}

func TestValidateContractHandshakeNonStrictWithoutPluginCall(t *testing.T) {
	rt := &Runtime{}
	require.NoError(t, rt.validateContractHandshake())

	rt.expectedHandshake = &runtimecontract.Handshake{}
	err := rt.validateContractHandshake()
	require.EqualError(
		t,
		err,
		"contract handshake requested, but plugin does not export __plugin_call",
	)
}

func TestValidateContractHandshakeFixtures(t *testing.T) {
	ctx := context.Background()
	baseSchema := runtimecontract.Schema{
		Name:       "handshake-test",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
	}

	t.Run("strict abi returns no results", func(t *testing.T) {
		plugin, err := New(
			ctx,
			WithFile(HANDSHAKE_NOABI, WithAllowUnsigned()),
			WithContractSchema(baseSchema),
		)
		require.Error(t, err)
		require.Nil(t, plugin)
		require.EqualError(t, err, "plugin __hookr_abi_version returned no results")
	})

	t.Run("strict missing handshake exports", func(t *testing.T) {
		plugin, err := New(
			ctx,
			WithFile(HANDSHAKE_NOHANDSHAKE, WithAllowUnsigned()),
			WithContractSchema(baseSchema),
		)
		require.Error(t, err)
		require.Nil(t, plugin)
		require.EqualError(
			t,
			err,
			"contract handshake requested, but plugin does not export handshake functions",
		)
	})

	t.Run("strict schema hash returns no results", func(t *testing.T) {
		plugin, err := New(
			ctx,
			WithFile(HANDSHAKE_NOHASH, WithAllowUnsigned()),
			WithContractSchema(baseSchema),
		)
		require.Error(t, err)
		require.Nil(t, plugin)
		require.EqualError(t, err, "plugin __hookr_schema_hash returned no results")
	})

	t.Run("strict schema hash length must match", func(t *testing.T) {
		plugin, err := New(
			ctx,
			WithFile(HANDSHAKE_BADHASHLEN, WithAllowUnsigned()),
			WithContractSchema(baseSchema),
		)
		require.Error(t, err)
		require.Nil(t, plugin)
		require.EqualError(t, err, "plugin schema hash has invalid length: got 31 want 32")
	})

	t.Run("strict schema hash pointer must be readable", func(t *testing.T) {
		plugin, err := New(
			ctx,
			WithFile(HANDSHAKE_BADHASHPTR, WithAllowUnsigned()),
			WithContractSchema(baseSchema),
		)
		require.Error(t, err)
		require.Nil(t, plugin)
		require.Contains(t, err.Error(), "schema_hash")
	})

	t.Run("strict schema requires methods export", func(t *testing.T) {
		plugin, err := New(
			ctx,
			WithFile(HANDSHAKE_NOMETHODS, WithAllowUnsigned()),
			WithContractSchema(baseSchema),
		)
		require.Error(t, err)
		require.Nil(t, plugin)
		require.EqualError(
			t,
			err,
			"contract schema validation requested, but plugin does not export __hookr_methods",
		)
	})

	t.Run("strict methods payload must be word aligned", func(t *testing.T) {
		plugin, err := New(
			ctx,
			WithFile(HANDSHAKE_BADMETHODS, WithAllowUnsigned()),
			WithContractSchema(baseSchema),
		)
		require.Error(t, err)
		require.Nil(t, plugin)
		require.EqualError(t, err, "plugin methods payload has invalid length: got 3")
	})

	t.Run("strict capabilities export must return a value", func(t *testing.T) {
		rt := mustNewStrictHandshakeRuntime(t, ctx, HANDSHAKE_CAPS_NORESULTS)
		err := rt.Instantiate()
		require.EqualError(t, err, "plugin __hookr_capabilities returned no results")
		require.NoError(t, rt.Close(ctx))
	})

	t.Run("strict capabilities export traps propagate as errors", func(t *testing.T) {
		rt := mustNewStrictHandshakeRuntime(t, ctx, HANDSHAKE_CAPS_TRAP)
		err := rt.Instantiate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to call __hookr_capabilities")
		require.NoError(t, rt.Close(ctx))
	})

	t.Run("non-strict abi errors are ignored", func(t *testing.T) {
		plugin, err := New(ctx, WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()))
		require.NoError(t, err)
		require.NoError(t, plugin.plugin.Close(ctx))
		t.Cleanup(func() {
			require.NoError(t, plugin.Close(ctx))
		})
		plugin.expectedHandshake = nil
		plugin.expectedSchema = nil
		require.NoError(t, plugin.validateContractHandshake())
	})

	t.Run("strict abi call errors are returned", func(t *testing.T) {
		plugin, err := New(ctx, WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()))
		require.NoError(t, err)
		require.NoError(t, plugin.plugin.Close(ctx))
		t.Cleanup(func() {
			require.NoError(t, plugin.Close(ctx))
		})
		plugin.expectedHandshake = &runtimecontract.Handshake{}
		err = plugin.validateContractHandshake()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to call __hookr_abi_version")
	})
}

func TestDefaultRuntime(t *testing.T) {
	rt, err := DefaultRuntime(context.Background())
	require.NoError(t, err)
	require.NotNil(t, rt)
	require.NoError(t, rt.Close(context.Background()))
}

func TestRuntimeHandshakeRequiresAllRequiredMethods(t *testing.T) {
	ctx := context.Background()
	schema := runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 99, Name: "Missing", RequestType: "bytes", ResponseType: "bytes"},
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
	require.Contains(t, err.Error(), "plugin does not implement required method Missing (99)")
}

func TestRuntimeHandshakeAllowsMissingOptionalMethods(t *testing.T) {
	ctx := context.Background()
	schema := runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{
				ID:           99,
				Name:         "OptionalMissing",
				RequestType:  "bytes",
				ResponseType: "bytes",
				Optional:     true,
			},
		},
	}

	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithContractSchema(schema),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, p.Close(ctx))
	}()
	require.False(t, p.HasPluginMethodID(99))
}

func TestLoadPluginMethods(t *testing.T) {
	ctx := context.Background()
	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, p.Close(ctx))
	}()

	methods, err := p.loadPluginMethods(p.plugin.ExportedFunction(fnMethods))
	require.NoError(t, err)
	require.Contains(t, methods, uint32(2))
	require.Contains(t, methods, uint32(3))
}

func TestLoadPluginMethodsErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("call failure on closed module", func(t *testing.T) {
		p, err := New(
			ctx,
			WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
			WithHostMethodFns(HostFnMethod(1, HelloByte)),
		)
		require.NoError(t, err)
		methodsFn := p.plugin.ExportedFunction(fnMethods)
		require.NotNil(t, methodsFn)
		require.NoError(t, p.plugin.Close(ctx))
		t.Cleanup(func() {
			require.NoError(t, p.Close(ctx))
		})

		_, err = p.loadPluginMethods(methodsFn)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to call __hookr_methods")
	})

	t.Run("empty methods payload", func(t *testing.T) {
		p, err := New(
			ctx,
			WithFile(HANDSHAKE_EMPTYMETHODS, WithAllowUnsigned()),
			WithContractSchema(runtimecontract.Schema{
				Name:       "empty-methods",
				SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
			}),
		)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, p.Close(ctx))
		}()
		require.Empty(t, p.PluginMethodIDs())
	})

	t.Run("methods export returns no results", func(t *testing.T) {
		p, err := New(
			ctx,
			WithFile(HANDSHAKE_METHODS_NORESULTS, WithAllowUnsigned()),
			WithContractSchema(runtimecontract.Schema{
				Name:       "methods-no-results",
				SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
			}),
		)
		require.Error(t, err)
		require.Nil(t, p)
		require.EqualError(t, err, "plugin __hookr_methods returned no results")
	})
}

func TestRuntimePackedValueHelpers(t *testing.T) {
	ptr, dataLen, err := unpackPtrLenU64((uint64(17) << 32) | 99)
	require.NoError(t, err)
	require.Equal(t, uint32(17), ptr)
	require.Equal(t, uint32(99), dataLen)

	major, minor, err := decodeABIVersion((uint64(2) << 16) | 7)
	require.NoError(t, err)
	require.Equal(t, uint16(2), major)
	require.Equal(t, uint16(7), minor)

	_, _, err = decodeABIVersion(uint64(math.MaxUint32) + 1)
	require.Error(t, err)
}

func TestInstantiateRequiresCompiledModule(t *testing.T) {
	ctx := context.Background()
	rt := &Runtime{
		ctx:        ctx,
		newRuntime: DefaultRuntime,
		stderr:     os.Stderr,
		stdout:     os.Stdout,
		rand:       rand.Reader,
		logger:     logger.Default,
	}
	require.NoError(t, rt.Init())
	defer func() {
		require.NoError(t, rt.Close(ctx))
	}()

	err := rt.Instantiate()
	require.EqualError(t, err, "plugin not compiled")
}

func TestCallWithInvokeContext2(t *testing.T) {
	ctx := context.Background()
	r, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, r.Close(ctx))
	}()

	ic := &invoke.Context{Operation: "sum"}
	results, err := r.callWithInvokeContext2(ctx, ic, r.pluginCall, 3, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Nil(t, r.currentInvoke())
}

func TestInvokeMethodWithResponsePropagatesCallbackError(t *testing.T) {
	ctx := context.Background()
	p, err := New(
		ctx,
		WithFile(SIMPLE_METHOD_WASM, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
	)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, p.Close(ctx))
	}()

	err = p.InvokeMethodWithResponse(ctx, 2, []byte("Steve"), func([]byte) error {
		return errors.New("callback failed")
	})
	require.EqualError(t, err, "callback failed")
}

func TestCloseReturnsRuntimeCloseError(t *testing.T) {
	rt := &Runtime{
		r: closeErrRuntime{err: errors.New("planned close failure")},
	}
	err := rt.Close(context.Background())
	require.EqualError(t, err, "error closing runtime: planned close failure")
}

func mustNewStrictHandshakeRuntime(t *testing.T, ctx context.Context, wasmPath string) *Runtime {
	t.Helper()
	file, err := NewFile(wasmPath, WithAllowUnsigned())
	require.NoError(t, err)
	rt := &Runtime{
		ctx:        ctx,
		file:       file,
		newRuntime: DefaultRuntime,
		stderr:     os.Stderr,
		stdout:     os.Stdout,
		rand:       rand.Reader,
		logger:     logger.Default,
		expectedHandshake: &runtimecontract.Handshake{
			ABIMajor:   runtimecontract.ABIVersionMajor,
			ABIMinor:   runtimecontract.ABIVersionMinor,
			SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		},
	}
	require.NoError(t, rt.Init())
	require.NoError(t, rt.Compile())
	return rt
}

type closeErrRuntime struct {
	err error
}

func (c closeErrRuntime) Instantiate(context.Context, []byte) (api.Module, error) {
	panic("unused")
}

func (c closeErrRuntime) InstantiateWithConfig(
	context.Context,
	[]byte,
	wazero.ModuleConfig,
) (api.Module, error) {
	panic("unused")
}

func (c closeErrRuntime) NewHostModuleBuilder(string) wazero.HostModuleBuilder {
	panic("unused")
}

func (c closeErrRuntime) CompileModule(context.Context, []byte) (wazero.CompiledModule, error) {
	panic("unused")
}

func (c closeErrRuntime) InstantiateModule(
	context.Context,
	wazero.CompiledModule,
	wazero.ModuleConfig,
) (api.Module, error) {
	panic("unused")
}

func (c closeErrRuntime) CloseWithExitCode(context.Context, uint32) error {
	return c.err
}

func (c closeErrRuntime) Module(string) api.Module {
	return nil
}

func (c closeErrRuntime) Close(context.Context) error {
	return c.err
}
