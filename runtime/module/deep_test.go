package module

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mopeyjellyfish/hookr/runtime/invoke"
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
	defer rt.Close(ctx)
	mod, err := New(ctx, rt, nil, nil, nil)
	if err != nil {
		t.Fatalf("new host module: %v", err)
	}
	defer mod.Close(ctx)
}

func TestHookrModuleDeepPaths(t *testing.T) {
	ctx, rt, guest := instantiateGuestModule(t)
	defer rt.Close(ctx)
	defer guest.Close(ctx)

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
	if results[0] == 0 {
		t.Fatal("expected host response len")
	}
	h.hostResponse(ctx, guest, []uint64{8})
	if got, _ := guest.Memory().Read(8, uint32(len(ic.HostResp))); string(got) != "ok:hello" {
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
	if results[0] == 0 {
		t.Fatal("expected host error len")
	}
	h.hostError(ctx, guest, []uint64{48})
	if got, _ := guest.Memory().Read(48, uint32(len(ic.HostErr.Error()))); string(got) != "host-fail" {
		t.Fatalf("unexpected host err bytes: %q", string(got))
	}
}

func TestHookrModuleLog(t *testing.T) {
	ctx, rt, guest := instantiateGuestModule(t)
	defer rt.Close(ctx)
	defer guest.Close(ctx)

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
