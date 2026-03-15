package main

import runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"

//go:wasmexport __plugin_call
func pluginCall(methodID uint32, payloadLen uint32) uint32 {
	_, _ = methodID, payloadLen
	return 1
}

//go:wasmexport __hookr_abi_version
func abiVersion() uint64 {
	return (uint64(runtimecontract.ABIVersionMajor) << 16) |
		uint64(runtimecontract.ABIVersionMinor)
}

//go:wasmexport __hookr_schema_hash
func schemaHash() {}
