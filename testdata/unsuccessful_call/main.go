package main

//go:wasmexport __plugin_call
func pluginCall(methodID uint32, payloadLen uint32) uint32 {
	return 0
}

func main() {}
