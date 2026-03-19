package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/stretchr/testify/require"
)

const SIMPLE_METHOD_RELOAD_WASM = "../testdata/simplemethod_reload/bin/simplemethod_reload.wasm"

func TestLiveRuntimeReloadsPluginAndBlocksCalls(t *testing.T) {
	ctx := context.Background()
	tempPlugin := copyWASMFixture(t, SIMPLE_METHOD_WASM)

	reloadStarted := make(chan struct{})
	reloadRelease := make(chan struct{})
	var reloadStartOnce sync.Once
	var reloadReleaseOnce sync.Once
	defer reloadReleaseOnce.Do(func() {
		close(reloadRelease)
	})

	rt, err := NewLive(ctx, ReloadConfig{
		Debounce: 25 * time.Millisecond,
		OnReload: func(ctx context.Context, next Invoker, event ReloadEvent) error {
			reloadStartOnce.Do(func() {
				close(reloadStarted)
			})
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

	rt.stopOnce.Do(func() {
		close(rt.stopCh)
	})
	<-rt.doneCh

	copyFixtureBytes(t, tempPlugin, SIMPLE_METHOD_RELOAD_WASM)

	reloadErr := make(chan error, 1)
	go func() {
		reloadErr <- rt.safeReloadNow()
	}()

	select {
	case <-reloadStarted:
	case <-time.After(10 * time.Second):
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

	reloadReleaseOnce.Do(func() {
		close(reloadRelease)
	})

	select {
	case err := <-callErr:
		t.Fatalf("call failed after reload: %v", err)
	case resp := <-callDone:
		require.GreaterOrEqual(t, time.Since(start), 75*time.Millisecond)
		require.Equal(t, "102", string(resp))
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for post-reload call")
	}

	select {
	case err := <-reloadErr:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reload to complete")
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

func TestLiveRuntimeCloseIsSafeConcurrently(t *testing.T) {
	ctx := context.Background()
	tempPlugin := copyWASMFixture(t, SIMPLE_METHOD_WASM)

	rt, err := NewLive(
		ctx,
		ReloadConfig{},
		WithFile(tempPlugin, WithAllowUnsigned()),
		WithContractSchema(simpleMethodSchema()),
	)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, rt.Close(ctx))
		}()
	}
	wg.Wait()
}

func TestLiveRuntimeOnReloadPanicReportsErrorAndKeepsWatching(t *testing.T) {
	ctx := context.Background()
	tempPlugin := copyWASMFixture(t, SIMPLE_METHOD_WASM)

	var panicked atomic.Bool
	reloadErrors := make(chan error, 2)
	rt, err := NewLive(ctx, ReloadConfig{
		Debounce: 25 * time.Millisecond,
		OnReload: func(ctx context.Context, next Invoker, event ReloadEvent) error {
			if panicked.CompareAndSwap(false, true) {
				panic("boom")
			}
			return nil
		},
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

	copyFixtureBytes(t, tempPlugin, SIMPLE_METHOD_RELOAD_WASM)

	select {
	case err := <-reloadErrors:
		require.Error(t, err)
		require.Contains(t, err.Error(), "reload hook panicked")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for reload panic")
	}

	copyFixtureBytes(t, tempPlugin, SIMPLE_METHOD_WASM)
	copyFixtureBytes(t, tempPlugin, SIMPLE_METHOD_RELOAD_WASM)

	require.Eventually(t, func() bool {
		resp, err := rt.InvokeMethod(ctx, 3, []byte("Steve"))
		return err == nil && string(resp) == "102"
	}, 3*time.Second, 50*time.Millisecond)
}

func TestLiveRuntimeClosePreviousRuntimeErrorReportsButKeepsSwap(t *testing.T) {
	ctx := context.Background()
	tempPlugin := copyWASMFixture(t, SIMPLE_METHOD_WASM)

	t.Cleanup(func() {
		closeReloadRuntime = func(ctx context.Context, rt *Runtime) error {
			return rt.Close(ctx)
		}
	})

	var injected atomic.Bool
	reloadErrors := make(chan error, 1)
	closeReloadRuntime = func(ctx context.Context, rt *Runtime) error {
		if injected.CompareAndSwap(false, true) {
			_ = rt.Close(ctx)
			return errors.New("forced close error")
		}
		return rt.Close(ctx)
	}

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

	copyFixtureBytes(t, tempPlugin, SIMPLE_METHOD_RELOAD_WASM)

	select {
	case err := <-reloadErrors:
		require.Error(t, err)
		require.Contains(t, err.Error(), "close previous runtime after reload")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for close error")
	}

	resp, err := rt.InvokeMethod(ctx, 3, []byte("Steve"))
	require.NoError(t, err)
	require.Equal(t, "102", string(resp))
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
