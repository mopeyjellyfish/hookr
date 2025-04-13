//go:build wasip1

package pdk

//go:wasm-module hookr
//go:export __plugin_request
func pluginRequest(operationPtr uintptr, payloadPtr uintptr)

//go:wasm-module hookr
//go:export __plugin_response
func pluginResponse(ptr uintptr, len uint32)

//go:wasm-module hookr
//go:export __plugin_error
func pluginError(ptr uintptr, len uint32)

//go:wasm-module hookr
//go:export __host_call
func hostCall(
	operationPtr uintptr, operationLen uint32,
	payloadPtr uintptr, payloadLen uint32) bool

//go:wasm-module hookr
//go:export __host_response_len
func hostResponseLen() uint32

//go:wasm-module hookr
//go:export __host_response
func hostResponse(ptr uintptr)

//go:wasm-module hookr
//go:export __host_error_len
func hostErrorLen() uint32

//go:wasm-module hookr
//go:export __host_error
func hostError(ptr uintptr)

//go:wasm-module hookr
//go:export __log
func consoleLog(ptr uintptr, len uint32)
