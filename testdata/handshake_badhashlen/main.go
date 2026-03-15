package main

import (
	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"unsafe"
)

var badSchemaHash = [31]byte{
	'h', 'o', 'o', 'k', 'r', '-', 's', 'i', 'm', 'p', 'l', 'e', '-', 'm', 'e', 't',
	'h', 'o', 'd', '-', 's', 'c', 'h', 'e', 'm', 'a', '-', '0', '0', '0',
}

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
func schemaHash() uint64 {
	ptr := uint32(uintptr(unsafe.Pointer(&badSchemaHash[0])))
	return (uint64(ptr) << 32) | uint64(len(badSchemaHash))
}
