package module

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mopeyjellyfish/hookr/runtime/invoke"
	runtimememory "github.com/mopeyjellyfish/hookr/runtime/memory"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const emptyWasmPath = "../../testdata/empty/bin/empty.wasm"

func instantiateGuestModule(t *testing.T) (context.Context, wazero.Runtime, api.Module) {
	t.Helper()
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		t.Fatalf("instantiate wasi: %v", err)
	}
	data, err := os.ReadFile(filepath.Clean(emptyWasmPath))
	if err != nil {
		t.Fatalf("read guest wasm: %v", err)
	}
	compiled, err := rt.CompileModule(ctx, data)
	if err != nil {
		t.Fatalf("compile guest wasm: %v", err)
	}
	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		t.Fatalf("instantiate guest wasm: %v", err)
	}
	return ctx, rt, mod
}

func cleanupClose(
	t *testing.T,
	ctx context.Context,
	name string,
	closeFn func(context.Context) error,
) {
	t.Helper()
	t.Cleanup(func() {
		if err := closeFn(ctx); err != nil {
			t.Errorf("close %s: %v", name, err)
		}
	})
}

func mustUint32FromLen(t *testing.T, n int) uint32 {
	t.Helper()
	v, err := runtimememory.Uint32FromInt(n)
	if err != nil {
		t.Fatalf("length conversion: %v", err)
	}
	return v
}

func TestInvokeContextResolution(t *testing.T) {
	ic := &invoke.Context{Operation: "one"}
	h := &hookrModule{currentInvoke: func() *invoke.Context { return ic }}
	if got := h.invokeContext(context.Background()); got != ic {
		t.Fatalf("expected current invoke context")
	}

	ctx := invoke.New(context.Background(), &invoke.Context{Operation: "two"})
	h = &hookrModule{}
	if got := h.invokeContext(ctx); got == nil || got.Operation != "two" {
		t.Fatalf("unexpected invoke context: %#v", got)
	}
}

func TestNewInstantiatesHookrModule(t *testing.T) {
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	cleanupClose(t, ctx, "runtime", rt.Close)
	mod, err := New(ctx, rt, nil, nil, nil)
	if err != nil {
		t.Fatalf("new host module: %v", err)
	}
	cleanupClose(t, ctx, "module", mod.Close)
}

func TestHookrModuleDeepPaths(t *testing.T) {
	ctx, rt, guest := instantiateGuestModule(t)
	cleanupClose(t, ctx, "runtime", rt.Close)
	cleanupClose(t, ctx, "guest", guest.Close)

	ic := &invoke.Context{PluginReq: []byte("hello")}
	h := &hookrModule{
		methodCallHandler: func(_ context.Context, methodID uint32, payload []byte) ([]byte, error) {
			if methodID == 7 {
				return append([]byte("ok:"), payload...), nil
			}
			return nil, errors.New("bad method")
		},
		currentInvoke: func() *invoke.Context { return ic },
		logger:        func(string) {},
	}

	if ok := guest.Memory().Write(0, make([]byte, 64)); !ok {
		t.Fatal("prepare guest memory")
	}
	if ok := guest.Memory().Write(0, []byte("hello")); !ok {
		t.Fatal("write payload")
	}

	stack := []uint64{7, 0, 5}
	h.hostCall(ctx, guest, stack)
	if stack[0] != 1 {
		t.Fatalf("expected successful host call, got %d", stack[0])
	}
	results := []uint64{0}
	h.hostResponseLen(ctx, results)
	if len(results) == 0 || results[0] == 0 {
		t.Fatal("expected host response len")
	}
	h.hostResponse(ctx, guest, []uint64{8})
	if got, _ := guest.Memory().Read(8, mustUint32FromLen(t, len(ic.HostResp))); string(
		got,
	) != "ok:hello" {
		t.Fatalf("unexpected host response bytes: %q", string(got))
	}

	h.pluginRequest(ctx, guest, []uint64{20})
	if got, _ := guest.Memory().Read(20, 5); string(got) != "hello" {
		t.Fatalf("unexpected plugin request bytes: %q", string(got))
	}

	if ok := guest.Memory().Write(30, []byte("resp")); !ok {
		t.Fatal("write plugin response")
	}
	h.pluginResponse(ctx, guest, []uint64{30, 4})
	if string(ic.PluginResp) != "resp" {
		t.Fatalf("unexpected plugin resp: %q", string(ic.PluginResp))
	}

	if ok := guest.Memory().Write(40, []byte("boom")); !ok {
		t.Fatal("write plugin error")
	}
	h.pluginError(ctx, guest, []uint64{40, 4})
	if ic.PluginErr != "boom" {
		t.Fatalf("unexpected plugin err: %q", ic.PluginErr)
	}

	ic.HostErr = errors.New("host-fail")
	h.hostErrorLen(ctx, results)
	if len(results) == 0 || results[0] == 0 {
		t.Fatal("expected host error len")
	}
	h.hostError(ctx, guest, []uint64{48})
	if got, _ := guest.Memory().Read(48, mustUint32FromLen(t, len(ic.HostErr.Error()))); string(
		got,
	) != "host-fail" {
		t.Fatalf("unexpected host err bytes: %q", string(got))
	}
}

func TestHookrModuleLog(t *testing.T) {
	ctx, rt, guest := instantiateGuestModule(t)
	cleanupClose(t, ctx, "runtime", rt.Close)
	cleanupClose(t, ctx, "guest", guest.Close)

	var messages []string
	h := &hookrModule{
		logger: func(msg string) {
			messages = append(messages, msg)
		},
	}
	if ok := guest.Memory().Write(0, []byte("hello-log")); !ok {
		t.Fatal("write log payload")
	}
	h.log(ctx, guest, []uint64{0, 9})
	if len(messages) != 1 || messages[0] != "hello-log" {
		t.Fatalf("unexpected log messages: %#v", messages)
	}
}

func TestHookrModuleErrorPaths(t *testing.T) {
	ctx, rt, guest := instantiateGuestModule(t)
	cleanupClose(t, ctx, "runtime", rt.Close)
	cleanupClose(t, ctx, "guest", guest.Close)

	ic := &invoke.Context{
		PluginReq: []byte("hello"),
		HostResp:  []byte("world"),
		HostErr:   errors.New("boom"),
	}
	h := &hookrModule{
		methodCallHandler: func(_ context.Context, methodID uint32, payload []byte) ([]byte, error) {
			return append([]byte("ok:"), payload...), nil
		},
		currentInvoke: func() *invoke.Context { return ic },
		logger:        func(string) {},
	}

	stack := []uint64{7, ^uint64(0), 5}
	h.hostCall(ctx, guest, stack)
	if stack[0] != 0 || ic.HostErr == nil {
		t.Fatalf(
			"expected failed host call with host error, got stack=%v err=%v",
			stack,
			ic.HostErr,
		)
	}

	ic.HostErr = nil
	h.pluginRequest(ctx, guest, []uint64{^uint64(0)})
	if ic.PluginErr == "" {
		t.Fatal("expected plugin request write failure")
	}

	ic.PluginErr = ""
	h.hostResponse(ctx, guest, []uint64{^uint64(0)})
	if ic.HostErr == nil {
		t.Fatal("expected host response write failure")
	}

	ic.HostErr = errors.New("boom")
	h.hostError(ctx, guest, []uint64{^uint64(0)})
	if ic.HostErr == nil {
		t.Fatal("expected host error write failure")
	}

	ic.PluginErr = ""
	h.pluginResponse(ctx, guest, []uint64{^uint64(0), 5})
	if ic.PluginErr == "" || ic.PluginResp != nil {
		t.Fatalf(
			"expected plugin response read failure, got err=%q resp=%v",
			ic.PluginErr,
			ic.PluginResp,
		)
	}

	ic.PluginErr = ""
	h.pluginError(ctx, guest, []uint64{^uint64(0), 4})
	if ic.PluginErr == "" {
		t.Fatal("expected plugin error read failure")
	}

	var messages []string
	h.logger = func(msg string) { messages = append(messages, msg) }
	h.log(ctx, guest, []uint64{^uint64(0), 5})
	if len(messages) != 1 || messages[0] == "" {
		t.Fatalf("expected logger to receive read failure, got %#v", messages)
	}
}

func TestHookrModuleLengthAndDispatchBranches(t *testing.T) {
	ctx, rt, guest := instantiateGuestModule(t)
	cleanupClose(t, ctx, "runtime", rt.Close)
	cleanupClose(t, ctx, "guest", guest.Close)

	t.Run("hostCall returns zero without invoke context", func(t *testing.T) {
		h := &hookrModule{
			methodCallHandler: func(_ context.Context, _ uint32, _ []byte) ([]byte, error) {
				t.Fatal("methodCallHandler should not be invoked")
				return nil, nil
			},
		}
		stack := []uint64{1, 0, 0}
		h.hostCall(ctx, guest, stack)
		if stack[0] != 0 {
			t.Fatalf("expected failed host call, got %d", stack[0])
		}
	})

	t.Run("hostCall returns zero without handler", func(t *testing.T) {
		h := &hookrModule{
			currentInvoke: func() *invoke.Context { return &invoke.Context{} },
		}
		stack := []uint64{1, 0, 0}
		h.hostCall(ctx, guest, stack)
		if stack[0] != 0 {
			t.Fatalf("expected failed host call, got %d", stack[0])
		}
	})

	t.Run("hostCall rejects oversized host response", func(t *testing.T) {
		restoreLenToU32 := stubLenToU32(t, 0, errors.New("too large"))
		defer restoreLenToU32()

		ic := &invoke.Context{}
		h := &hookrModule{
			currentInvoke: func() *invoke.Context { return ic },
			methodCallHandler: func(_ context.Context, _ uint32, _ []byte) ([]byte, error) {
				return []byte("resp"), nil
			},
		}
		stack := []uint64{1, 0, 0}
		h.hostCall(ctx, guest, stack)
		if stack[0] != 0 {
			t.Fatalf("expected failed host call, got %d", stack[0])
		}
		if ic.HostErr == nil || ic.HostResp != nil {
			t.Fatalf(
				"expected oversized host response error, got err=%v resp=%v",
				ic.HostErr,
				ic.HostResp,
			)
		}
	})

	t.Run("hostCall normalizes oversized host error message", func(t *testing.T) {
		restoreLenToU32 := stubLenToU32(t, 0, errors.New("too large"))
		defer restoreLenToU32()

		ic := &invoke.Context{}
		h := &hookrModule{
			currentInvoke: func() *invoke.Context { return ic },
			methodCallHandler: func(_ context.Context, _ uint32, _ []byte) ([]byte, error) {
				return nil, errors.New("boom")
			},
		}
		stack := []uint64{1, 0, 0}
		h.hostCall(ctx, guest, stack)
		if stack[0] != 0 {
			t.Fatalf("expected failed host call, got %d", stack[0])
		}
		if ic.HostErr == nil || ic.HostErr.Error() != "host error message too large" {
			t.Fatalf("unexpected host error: %v", ic.HostErr)
		}
	})

	t.Run("hostResponseLen handles nil invoke and nil response", func(t *testing.T) {
		h := &hookrModule{}
		results := []uint64{99}
		h.hostResponseLen(ctx, results)
		if results[0] != 0 {
			t.Fatalf("expected zero result for nil invoke context, got %d", results[0])
		}

		h.currentInvoke = func() *invoke.Context { return &invoke.Context{} }
		results[0] = 99
		h.hostResponseLen(ctx, results)
		if results[0] != 0 {
			t.Fatalf("expected zero result without host response, got %d", results[0])
		}
	})

	t.Run("hostResponseLen rejects oversized response", func(t *testing.T) {
		restoreLenToU32 := stubLenToU32(t, 0, errors.New("too large"))
		defer restoreLenToU32()

		ic := &invoke.Context{HostResp: []byte("resp")}
		h := &hookrModule{currentInvoke: func() *invoke.Context { return ic }}
		results := []uint64{99}
		h.hostResponseLen(ctx, results)
		if results[0] != 0 {
			t.Fatalf("expected zero result, got %d", results[0])
		}
		if ic.HostErr == nil || ic.HostResp != nil {
			t.Fatalf(
				"expected oversized response error, got err=%v resp=%v",
				ic.HostErr,
				ic.HostResp,
			)
		}
	})

	t.Run("hostErrorLen handles nil invoke and nil error", func(t *testing.T) {
		h := &hookrModule{}
		results := []uint64{99}
		h.hostErrorLen(ctx, results)
		if results[0] != 0 {
			t.Fatalf("expected zero result for nil invoke context, got %d", results[0])
		}

		h.currentInvoke = func() *invoke.Context { return &invoke.Context{} }
		results[0] = 99
		h.hostErrorLen(ctx, results)
		if results[0] != 0 {
			t.Fatalf("expected zero result without host error, got %d", results[0])
		}
	})

	t.Run("hostErrorLen rejects oversized error message", func(t *testing.T) {
		restoreLenToU32 := stubLenToU32(t, 0, errors.New("too large"))
		defer restoreLenToU32()

		ic := &invoke.Context{HostErr: errors.New("boom")}
		h := &hookrModule{currentInvoke: func() *invoke.Context { return ic }}
		results := []uint64{99}
		h.hostErrorLen(ctx, results)
		if results[0] != 0 {
			t.Fatalf("expected zero result, got %d", results[0])
		}
	})
}

func stubLenToU32(t *testing.T, value uint32, err error) func() {
	t.Helper()
	previous := lenToU32
	lenToU32 = func(int) (uint32, error) {
		return value, err
	}
	return func() {
		lenToU32 = previous
	}
}
