package module

import (
	"context"

	"github.com/mopeyjellyfish/hookr/runtime/invoke"
	"github.com/mopeyjellyfish/hookr/runtime/logger"
	"github.com/mopeyjellyfish/hookr/runtime/memory"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const i32 = api.ValueTypeI32

// CallHandler is a function to invoke to handle when a guest is performing a host call.
type CallHandler func(ctx context.Context, operation string, payload []byte) ([]byte, error)

// CallHandlerV2 handles host callbacks by numeric method ID.
type CallHandlerV2 func(ctx context.Context, methodID uint32, payload []byte) ([]byte, error)

// hookrModule implements all required hookr host function exports.
type hookrModule struct {
	// callHandler implements hostCall, which returns false (0) when nil.
	callHandler CallHandler
	// callHandlerV2 implements hostCallV2, which returns false (0) when nil.
	callHandlerV2 CallHandlerV2
	// currentInvoke returns the active invoke context without requiring context.WithValue.
	currentInvoke func() *invoke.Context

	// logger is used to implement consoleLog.
	logger logger.Logger
}

// instantiateHookrHost instantiates a hookrModule and returns it and its corresponding module, or an error.
//   - r: used to instantiate the hookr host module
//   - callHandler: used to implement hostCall
//   - logger: used to implement consoleLog
func instantiateHookrModule(
	ctx context.Context,
	r wazero.Runtime,
	callHandler CallHandler,
	callHandlerV2 CallHandlerV2,
	currentInvoke func() *invoke.Context,
	logger logger.Logger,
) (api.Module, error) {
	h := &hookrModule{
		callHandler:   callHandler,
		callHandlerV2: callHandlerV2,
		currentInvoke: currentInvoke,
		logger:        logger,
	}
	return r.NewHostModuleBuilder("hookr").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.hostCall), []api.ValueType{i32, i32, i32, i32}, []api.ValueType{i32}).
		WithParameterNames("cmd_ptr", "cmd_len", "payload_ptr", "payload_len").
		Export("__host_call").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.hostCallV2), []api.ValueType{i32, i32, i32}, []api.ValueType{i32}).
		WithParameterNames("method_id", "payload_ptr", "payload_len").
		Export("__host_call_v2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.log), []api.ValueType{i32, i32}, []api.ValueType{}).
		WithParameterNames("ptr", "len").
		Export("__log").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.pluginRequest), []api.ValueType{i32, i32}, []api.ValueType{}).
		WithParameterNames("op_ptr", "ptr").
		Export("__plugin_request").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(h.pluginRequestV2), []api.ValueType{i32}, []api.ValueType{}).
		WithParameterNames("ptr").
		Export("__plugin_request_v2").
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

// hostCall is the WebAssembly function export "__host_call", which initiates a host using the callHandler using
// parameters read from linear memory (wasm.Memory).
func (w *hookrModule) hostCall(ctx context.Context, m api.Module, stack []uint64) {
	cmdPtr := api.DecodeU32(stack[0])
	cmdLen := api.DecodeU32(stack[1])
	payloadPtr := api.DecodeU32(stack[2])
	payloadLen := api.DecodeU32(stack[3])
	ic := w.invokeContext(ctx)
	if ic == nil || w.callHandler == nil {
		stack[0] = 0 // false: neither an invocation context, nor a callHandler
		return
	}

	mem := m.Memory()
	operation := memory.ReadString(mem, "operation", cmdPtr, cmdLen)
	payload := memory.Read(mem, "payload", payloadPtr, payloadLen)

	if ic.HostResp, ic.HostErr = w.callHandler(ctx, operation, payload); ic.HostErr != nil {
		stack[0] = 0 // false: error (assumed to be logged already?)
	} else {
		stack[0] = 1 // true
	}
}

// hostCallV2 is the WebAssembly function export "__host_call_v2", which performs
// method-ID based host callbacks.
func (w *hookrModule) hostCallV2(ctx context.Context, m api.Module, stack []uint64) {
	methodID := api.DecodeU32(stack[0])
	payloadPtr := api.DecodeU32(stack[1])
	payloadLen := api.DecodeU32(stack[2])
	ic := w.invokeContext(ctx)
	if ic == nil || w.callHandlerV2 == nil {
		stack[0] = 0
		return
	}

	payload := memory.Read(m.Memory(), "payload", payloadPtr, payloadLen)
	if ic.HostResp, ic.HostErr = w.callHandlerV2(ctx, methodID, payload); ic.HostErr != nil {
		stack[0] = 0
	} else {
		stack[0] = 1
	}
}

// consoleLog is the WebAssembly function export "__console_log", which logs the message stored by the guest at the
// given offset (ptr) and length (len) in linear memory (wasm.Memory).
func (w *hookrModule) log(_ context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	msgLen := api.DecodeU32(params[1])

	if log := w.logger; log != nil {
		msg := memory.ReadString(m.Memory(), "msg", ptr, msgLen)
		w.logger(msg)
	}
}

// pluginRequest is the WebAssembly function export "__plugin_request", which writes the invokeContext.operation and
// invokeContext.guestReq to the given offsets (opPtr, ptr) in linear memory (wasm.Memory).
func (w *hookrModule) pluginRequest(ctx context.Context, m api.Module, params []uint64) {
	opPtr := api.DecodeU32(params[0])
	ptr := api.DecodeU32(params[1])

	ic := w.invokeContext(ctx)
	if ic == nil {
		return // no invoke context
	}

	mem := m.Memory()
	if operation := ic.Operation; operation != "" {
		memory.Write(mem, "operation", opPtr, []byte(operation))
	}
	if guestReq := ic.PluginReq; guestReq != nil {
		memory.Write(mem, "guestReq", ptr, guestReq)
	}
}

// pluginRequestV2 is the WebAssembly function export "__plugin_request_v2", which
// writes only the current request payload for method-ID based calls.
func (w *hookrModule) pluginRequestV2(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])

	ic := w.invokeContext(ctx)
	if ic == nil {
		return
	}

	if guestReq := ic.PluginReq; guestReq != nil {
		memory.Write(m.Memory(), "guestReq", ptr, guestReq)
	}
}

// hostResponse is the WebAssembly function export "__host_response", which writes the invokeContext.hostResp to the
// given offset (ptr) in linear memory (wasm.Memory).
func (w *hookrModule) hostResponse(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])

	if ic := w.invokeContext(ctx); ic == nil {
		return // no invoke context
	} else if hostResp := ic.HostResp; hostResp != nil {
		memory.Write(m.Memory(), "hostResp", ptr, hostResp)
	}
}

// hostResponse is the WebAssembly function export "__host_response_len", which returns the length of the current host
// response from invokeContext.hostResp.
func (w *hookrModule) hostResponseLen(ctx context.Context, results []uint64) {
	if ic := w.invokeContext(ctx); ic == nil {
		results[0] = 0 // no invoke context
	} else if hostResp := ic.HostResp; hostResp != nil {
		hostResponseLen, err := memory.Uint32FromInt(len(hostResp))
		if err != nil {
			panic(err)
		}
		results[0] = uint64(hostResponseLen)
	} else {
		results[0] = 0 // no host response
	}
}

// pluginResponse is the WebAssembly function export "__plugin_response", which reads invokeContext.guestResp from the
// given offset (ptr) and length (len) in linear memory (wasm.Memory).
func (w *hookrModule) pluginResponse(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	dataLen := api.DecodeU32(params[1])

	if ic := w.invokeContext(ctx); ic == nil {
		return // no invoke context
	} else {
		ic.PluginResp = memory.Read(m.Memory(), "guestResp", ptr, dataLen)
	}
}

// pluginError is the WebAssembly function export "__plugin_error", which reads invokeContext.guestErr from the given
// offset (ptr) and length (len) in linear memory (wasm.Memory).
func (w *hookrModule) pluginError(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	errLen := api.DecodeU32(params[1])

	if ic := w.invokeContext(ctx); ic == nil {
		return // no invoke context
	} else {
		ic.PluginErr = memory.ReadString(m.Memory(), "guestErr", ptr, errLen)
	}
}

// hostError is the WebAssembly function export "__host_error", which writes the invokeContext.hostErr to the given
// offset (ptr) in linear memory (wasm.Memory).
func (w *hookrModule) hostError(ctx context.Context, m api.Module, params []uint64) {
	ptr := api.DecodeU32(params[0])
	if ic := w.invokeContext(ctx); ic == nil {
		return // no invoke context
	} else if hostErr := ic.HostErr; hostErr != nil {
		memory.Write(m.Memory(), "hostErr", ptr, []byte(hostErr.Error()))
	}
}

// hostError is the WebAssembly function export "__host_error_len", which returns the length of the current host error
// from invokeContext.hostErr.
func (w *hookrModule) hostErrorLen(ctx context.Context, results []uint64) {
	if ic := w.invokeContext(ctx); ic == nil {
		results[0] = 0 // no invoke context
	} else if hostErr := ic.HostErr; hostErr != nil {
		errorMsg := hostErr.Error()
		hostErrorLen, err := memory.Uint32FromInt(len(errorMsg))
		if err != nil {
			panic(err)
		}
		results[0] = uint64(hostErrorLen)
	} else {
		results[0] = 0 // no host error
	}
}

func New(
	ctx context.Context,
	rt wazero.Runtime,
	callHandler CallHandler,
	callHandlerV2 CallHandlerV2,
	currentInvoke func() *invoke.Context,
	logger logger.Logger,
) (api.Module, error) {
	return instantiateHookrModule(ctx, rt, callHandler, callHandlerV2, currentInvoke, logger)
}

func (w *hookrModule) invokeContext(ctx context.Context) *invoke.Context {
	if w.currentInvoke != nil {
		if ic := w.currentInvoke(); ic != nil {
			return ic
		}
	}
	return invoke.From(ctx)
}
