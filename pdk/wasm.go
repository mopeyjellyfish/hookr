//go:build wasip1

package pdk

//go:wasmimport hookr __plugin_request
func pluginRequest(payloadPtr uintptr) bool

//go:wasmimport hookr __plugin_response
func pluginResponse(ptr uintptr, len uint32)

//go:wasmimport hookr __plugin_error
func pluginError(ptr uintptr, len uint32)

//go:wasmimport hookr __host_call
func hostCall(
	methodID uint32,
	payloadPtr uintptr, payloadLen uint32,
) bool

//go:wasmimport hookr __host_response_len
func hostResponseLen() uint32

//go:wasmimport hookr __host_response
func hostResponse(ptr uintptr)

//go:wasmimport hookr __host_error_len
func hostErrorLen() uint32

//go:wasmimport hookr __host_error
func hostError(ptr uintptr)

//go:wasmimport hookr __log
func consoleLog(ptr uintptr, len uint32)
