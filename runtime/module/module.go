package module

import (
	"context"
	"errors"
	"fmt"

	"github.com/mopeyjellyfish/hookr/runtime/invoke"
	"github.com/mopeyjellyfish/hookr/runtime/logger"
	"github.com/mopeyjellyfish/hookr/runtime/memory"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const i32 = api.ValueTypeI32

// MethodCallHandler handles host callbacks by numeric method ID.
type MethodCallHandler func(ctx context.Context, methodID uint32, payload []byte) ([]byte, error)

// hookrModule implements the required Hookr host function exports.
type hookrModule struct {
	methodCallHandler MethodCallHandler
	currentInvoke     func() *invoke.Context
	logger            logger.Logger
}

func instantiateHookrModule(
	ctx context.Context,
	r wazero.Runtime,
	methodCallHandler MethodCallHandler,
	currentInvoke func() *invoke.Context,
	logFn logger.Logger,
) (api.Module, error) {
	h := &hookrModule{
		methodCallHandler: methodCallHandler,
		currentInvoke:     currentInvoke,
		logger:            logFn,
	}
	return r.NewHostModuleBuilder("hookr").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.hostCall), []api.ValueType{i32, i32, i32}, []api.ValueType{i32}).
		WithParameterNames("method_id", "payload_ptr", "payload_len").
		Export("__host_call").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.log), []api.ValueType{i32, i32}, []api.ValueType{}).
		WithParameterNames("ptr", "len").
		Export("__log").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.pluginRequest), []api.ValueType{i32}, []api.ValueType{i32}).
		WithParameterNames("ptr").
		Export("__plugin_request").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.hostResponse), []api.ValueType{i32}, []api.ValueType{}).
		WithParameterNames("ptr").
		Export("__host_response").
		NewFunctionBuilder().
		WithGoFunction(api.GoFunc(h.hostResponseLen), []api.ValueType{}, []api.ValueType{i32}).
		Export("__host_response_len").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.pluginResponse), []api.ValueType{i32, i32}, []api.ValueType{}).
		WithParameterNames("ptr", "len").
		Export("__plugin_response").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.pluginError), []api.ValueType{i32, i32}, []api.ValueType{}).
		WithParameterNames("ptr", "len").
		Export("__plugin_error").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.hostError), []api.ValueType{i32}, []api.ValueType{}).
		WithParameterNames("ptr").
		Export("__host_error").
		NewFunctionBuilder().
		WithGoFunction(api.GoFunc(h.hostErrorLen), []api.ValueType{}, []api.ValueType{i32}).
		Export("__host_error_len").
		Instantiate(ctx)
}

// hostCall is the WebAssembly export "__host_call", which performs method-ID based host callbacks.
func (w *hookrModule) hostCall(ctx context.Context, m api.Module, stack []uint64) {
	methodID := api.DecodeU32(stack[0])
	payloadPtr := api.DecodeU32(stack[1])
	payloadLen := api.DecodeU32(stack[2])
	ic := w.invokeContext(ctx)
	if ic == nil || w.methodCallHandler == nil {
		stack[0] = 0
		return
	}

	payload, err := memory.TryRead(m.Memory(), "payload", payloadPtr, payloadLen)
	if err != nil {
		ic.HostErr = err
		stack[0] = 0
		return
	}
	if ic.HostResp, ic.HostErr = w.methodCallHandler(ctx, methodID, payload); ic.HostErr != nil {
		if _, err := memory.Uint32FromInt(len(ic.HostErr.Error())); err != nil {
			ic.HostErr = errors.New("host error message too large")
		}
		stack[0] = 0
	} else if _, err := memory.Uint32FromInt(len(ic.HostResp)); err != nil {
		ic.HostErr = fmt.Errorf("host response too large: %w", err)
		ic.HostResp = nil
		stack[0] = 0
	} else {
		stack[0] = 1
	}
}

func (w *hookrModule) log(_ context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	msgLen := api.DecodeU32(params[1])

	if log := w.logger; log != nil {
		msg, err := memory.TryReadString(m.Memory(), "msg", ptr, msgLen)
		if err != nil {
			w.logger(err.Error())
			return
		}
		w.logger(msg)
	}
}

// pluginRequest writes the current request payload for method-ID based calls.
func (w *hookrModule) pluginRequest(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])

	ic := w.invokeContext(ctx)
	if ic == nil {
		params[0] = 0
		return
	}

	if guestReq := ic.PluginReq; guestReq != nil {
		if err := memory.TryWrite(m.Memory(), "guestReq", ptr, guestReq); err != nil {
			ic.PluginErr = err.Error()
			params[0] = 0
			return
		}
	}
	params[0] = 1
}

func (w *hookrModule) hostResponse(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])

	if ic := w.invokeContext(ctx); ic == nil {
		return
	} else if hostResp := ic.HostResp; hostResp != nil {
		if err := memory.TryWrite(m.Memory(), "hostResp", ptr, hostResp); err != nil {
			ic.HostErr = err
		}
	}
}

func (w *hookrModule) hostResponseLen(ctx context.Context, results []uint64) {
	if ic := w.invokeContext(ctx); ic == nil {
		results[0] = 0
	} else if hostResp := ic.HostResp; hostResp != nil {
		hostResponseLen, err := memory.Uint32FromInt(len(hostResp))
		if err != nil {
			ic.HostErr = fmt.Errorf("host response too large: %w", err)
			ic.HostResp = nil
			results[0] = 0
			return
		}
		results[0] = uint64(hostResponseLen)
	} else {
		results[0] = 0
	}
}

func (w *hookrModule) pluginResponse(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	dataLen := api.DecodeU32(params[1])

	if ic := w.invokeContext(ctx); ic == nil {
		return
	} else {
		resp, err := memory.TryRead(m.Memory(), "guestResp", ptr, dataLen)
		if err != nil {
			ic.PluginErr = err.Error()
			ic.PluginResp = nil
			return
		}
		ic.PluginResp = resp
	}
}

func (w *hookrModule) pluginError(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	errLen := api.DecodeU32(params[1])

	if ic := w.invokeContext(ctx); ic == nil {
		return
	} else {
		msg, err := memory.TryReadString(m.Memory(), "guestErr", ptr, errLen)
		if err != nil {
			ic.PluginErr = err.Error()
			return
		}
		ic.PluginErr = msg
	}
}

func (w *hookrModule) hostError(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	if ic := w.invokeContext(ctx); ic == nil {
		return
	} else if hostErr := ic.HostErr; hostErr != nil {
		if err := memory.TryWrite(m.Memory(), "hostErr", ptr, []byte(hostErr.Error())); err != nil {
			ic.HostErr = err
		}
	}
}

func (w *hookrModule) hostErrorLen(ctx context.Context, results []uint64) {
	if ic := w.invokeContext(ctx); ic == nil {
		results[0] = 0
	} else if hostErr := ic.HostErr; hostErr != nil {
		errorMsg := hostErr.Error()
		hostErrorLen, err := memory.Uint32FromInt(len(errorMsg))
		if err != nil {
			results[0] = 0
			return
		}
		results[0] = uint64(hostErrorLen)
	} else {
		results[0] = 0
	}
}

func New(
	ctx context.Context,
	rt wazero.Runtime,
	methodCallHandler MethodCallHandler,
	currentInvoke func() *invoke.Context,
	logFn logger.Logger,
) (api.Module, error) {
	return instantiateHookrModule(ctx, rt, methodCallHandler, currentInvoke, logFn)
}

func (w *hookrModule) invokeContext(ctx context.Context) *invoke.Context {
	if w.currentInvoke != nil {
		if ic := w.currentInvoke(); ic != nil {
			return ic
		}
	}
	return invoke.From(ctx)
}
