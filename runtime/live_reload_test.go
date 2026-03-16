package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/stretchr/testify/require"
)

const SIMPLE_METHOD_RELOAD_WASM = "../testdata/simplemethod_reload/bin/simplemethod_reload.wasm"

func TestLiveRuntimeReloadsPluginAndBlocksCalls(t *testing.T) {
	ctx := context.Background()
	tempPlugin := copyWASMFixture(t, SIMPLE_METHOD_WASM)

	reloadStarted := make(chan struct{}, 1)
	reloadRelease := make(chan struct{})

	rt, err := NewLive(ctx, ReloadConfig{
		Debounce: 25 * time.Millisecond,
		OnReload: func(ctx context.Context, next Invoker, event ReloadEvent) error {
			select {
			case reloadStarted <- struct{}{}:
			default:
			}
			<-reloadRelease
			return nil
		},
	}, WithFile(tempPlugin, WithAllowUnsigned()), WithContractSchema(simpleMethodSchema()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rt.Close(ctx))
	}()

	before, err := rt.InvokeMethod(ctx, 3, []byte("Steve"))
	require.NoError(t, err)
	require.Equal(t, "2", string(before))

	copyFixtureBytes(t, tempPlugin, SIMPLE_METHOD_RELOAD_WASM)

	select {
	case <-reloadStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for reload to start")
	}

	start := time.Now()
	callDone := make(chan []byte, 1)
	callErr := make(chan error, 1)
	go func() {
		resp, err := rt.InvokeMethod(ctx, 3, []byte("Steve"))
		if err != nil {
			callErr <- err
			return
		}
		callDone <- resp
	}()

	select {
	case <-callDone:
		t.Fatal("call should have blocked while reload was in progress")
	case err := <-callErr:
		t.Fatalf("call failed while waiting for reload: %v", err)
	case <-time.After(75 * time.Millisecond):
	}

	close(reloadRelease)

	select {
	case err := <-callErr:
		t.Fatalf("call failed after reload: %v", err)
	case resp := <-callDone:
		require.GreaterOrEqual(t, time.Since(start), 75*time.Millisecond)
		require.Equal(t, "102", string(resp))
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for post-reload call")
	}
}

func TestLiveRuntimeKeepsPreviousRuntimeOnReloadFailure(t *testing.T) {
	ctx := context.Background()
	tempPlugin := copyWASMFixture(t, SIMPLE_METHOD_WASM)

	reloadErrors := make(chan error, 1)
	rt, err := NewLive(ctx, ReloadConfig{
		Debounce: 25 * time.Millisecond,
		OnReloadError: func(ctx context.Context, err error) {
			select {
			case reloadErrors <- err:
			default:
			}
		},
	}, WithFile(tempPlugin, WithAllowUnsigned()), WithContractSchema(simpleMethodSchema()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rt.Close(ctx))
	}()

	copyFixtureBytes(t, tempPlugin, EMPTY_WASM)

	select {
	case err := <-reloadErrors:
		require.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for reload failure")
	}

	resp, err := rt.InvokeMethod(ctx, 3, []byte("Steve"))
	require.NoError(t, err)
	require.Equal(t, "2", string(resp))
}

func TestLiveRuntimeOnReloadErrorAbortsSwap(t *testing.T) {
	ctx := context.Background()
	tempPlugin := copyWASMFixture(t, SIMPLE_METHOD_WASM)

	hookCalled := make(chan struct{}, 1)
	rt, err := NewLive(ctx, ReloadConfig{
		Debounce: 25 * time.Millisecond,
		OnReload: func(ctx context.Context, next Invoker, event ReloadEvent) error {
			select {
			case hookCalled <- struct{}{}:
			default:
			}
			return context.Canceled
		},
	}, WithFile(tempPlugin, WithAllowUnsigned()), WithContractSchema(simpleMethodSchema()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rt.Close(ctx))
	}()

	copyFixtureBytes(t, tempPlugin, SIMPLE_METHOD_RELOAD_WASM)

	select {
	case <-hookCalled:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for reload hook")
	}

	resp, err := rt.InvokeMethod(ctx, 3, []byte("Steve"))
	require.NoError(t, err)
	require.Equal(t, "2", string(resp))
}

func copyWASMFixture(t *testing.T, source string) string {
	t.Helper()

	dir := t.TempDir()
	target := filepath.Join(dir, "plugin.wasm")
	copyFixtureBytes(t, target, source)
	return target
}

func copyFixtureBytes(t *testing.T, target, source string) {
	t.Helper()

	//nolint:gosec // Test fixture paths are controlled by the test.
	data, err := os.ReadFile(source)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(target, data, 0o600))
}

func simpleMethodSchema() runtimecontract.Schema {
	return runtimecontract.Schema{
		Name:       "simple-method",
		SchemaHash: SIMPLE_METHOD_SCHEMA_HASH,
		Methods: []runtimecontract.Method{
			{ID: 2, Name: "Echo", RequestType: "bytes", ResponseType: "bytes"},
			{ID: 3, Name: "Vowel", RequestType: "bytes", ResponseType: "bytes"},
		},
	}
}
