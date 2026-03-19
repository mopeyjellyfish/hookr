package runtime

import (
	"context"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/stretchr/testify/require"
)

func TestStandardGoPluginBuildExportsAndRuns(t *testing.T) {
	t.Parallel()

	wasmPath := buildStandardGoPlugin(t, "../testdata/simplemethod")
	assertStandardGoPluginExports(t, wasmPath)

	ctx := context.Background()
	schema := runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}

	rt, err := New(
		ctx,
		WithFile(wasmPath, WithAllowUnsigned()),
		WithHostMethodFns(HostFnMethod(1, HelloByte)),
		WithContractSchema(schema),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, rt.Close(ctx))
	})

	require.True(t, rt.SupportsMethodABI())
	require.Equal(t, []uint32{2, 3}, rt.PluginMethodIDs())

	resp, err := rt.InvokeMethod(ctx, 2, []byte("Steve"))
	require.NoError(t, err)
	require.Equal(t, "Hello Steve", string(resp))
}

func buildStandardGoPlugin(t *testing.T, relPluginPath string) string {
	t.Helper()

	_, filename, _, ok := goruntime.Caller(0)
	require.True(t, ok)

	pluginPath := filepath.Join(filepath.Dir(filename), relPluginPath)
	outPath := filepath.Join(t.TempDir(), "plugin.wasm")
	cfg := buildkit.DefaultConfig()
	cfg.PluginPath = pluginPath
	cfg.OutputPath = outPath
	require.NoError(t, buildkit.Build(cfg))

	info, err := os.Stat(outPath)
	require.NoError(t, err)
	require.NotZero(t, info.Size())

	return outPath
}

func assertStandardGoPluginExports(t *testing.T, wasmPath string) {
	t.Helper()

	ctx := context.Background()
	r := newTestWazeroRuntime(ctx)
	t.Cleanup(func() {
		require.NoError(t, r.Close(ctx))
	})

	// #nosec G304 -- test controls the temporary wasm artifact path.
	wasm, err := os.ReadFile(wasmPath)
	require.NoError(t, err)

	compiled, err := r.CompileModule(ctx, wasm)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, compiled.Close(ctx))
	})

	exports := compiled.ExportedFunctions()
	for _, name := range []string{
		"_initialize",
		"hookr_init",
		"__plugin_call",
		"__hookr_abi_version",
		"__hookr_schema_hash",
		"__hookr_capabilities",
		"__hookr_methods",
	} {
		_, ok := exports[name]
		require.Truef(t, ok, "missing export %s", name)
	}
	_, hasStart := exports["_start"]
	require.False(t, hasStart, "c-shared plugin should export _initialize instead of _start")
}
