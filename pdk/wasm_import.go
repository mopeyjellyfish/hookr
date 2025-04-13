//go:build !wasip1

package pdk

func pluginRequest(operationPtr uintptr, payloadPtr uintptr) {

}

func pluginResponse(ptr uintptr, len uint32) {}

func pluginError(ptr uintptr, len uint32) {

}

func hostCall(
	operationPtr uintptr, operationLen uint32,
	payloadPtr uintptr, payloadLen uint32) bool {
	return false
}

//go:wasm-module hookr
//go:export __host_response_len
func hostResponseLen() uint32 {
	return 0
}

//go:wasm-module hookr
//go:export __host_response
func hostResponse(ptr uintptr) {}

//go:wasm-module hookr
//go:export __host_error_len
func hostErrorLen() uint32 {
	return 0
}

//go:wasm-module hookr
//go:export __host_error
func hostError(ptr uintptr) {}

//go:wasm-module hookr
//go:export __log
func consoleLog(ptr uintptr, len uint32) {}
