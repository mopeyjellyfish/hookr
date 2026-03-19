package main

//go:wasmexport __plugin_call
func pluginCall(methodID uint32, payloadLen uint32) uint32 {
	_, _ = methodID, payloadLen
	return 1
}

func main() {}
