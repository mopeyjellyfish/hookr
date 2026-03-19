//go:build !wasip1

package pdk

func pluginRequest(payloadPtr uintptr) bool { return false }

func pluginResponse(ptr uintptr, len uint32) {}

func pluginError(ptr uintptr, len uint32) {
}

func hostCall(
	methodID uint32,
	payloadPtr uintptr, payloadLen uint32,
) bool {
	return false
}

func hostResponseLen() uint32 {
	return 0
}

func hostResponse(ptr uintptr) {}

func hostErrorLen() uint32 {
	return 0
}

func hostError(ptr uintptr) {}

func consoleLog(ptr uintptr, len uint32) {}
