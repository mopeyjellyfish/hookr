package pdk

import (
	"encoding/binary"
	"fmt"
	"slices"
	"unsafe"
)

type HostError struct {
	message string
}

var methodDispatcher func(methodID uint32, payload []byte) ([]byte, error)

const (
	abiMajor         uint16 = 2
	abiMinor         uint16 = 0
	abiSchemaHashLen        = 32
)

var (
	abiSchemaHash    [abiSchemaHashLen]byte
	abiSchemaHashSet bool
	abiCapabilities  uint64
	abiMethods       []uint32
	abiMethodsRaw    []byte

	pluginPayloadScratch []byte
	hostRespScratch      []byte
	hostErrScratch       []byte
)

// SetMethodDispatcher installs a fast path dispatcher for method-ID plugin calls.
// Generated code can provide a switch-based dispatcher to avoid map lookups.
func SetMethodDispatcher(dispatcher func(methodID uint32, payload []byte) ([]byte, error)) {
	methodDispatcher = dispatcher
}

// SetABISchemaHash enables schema-handshake exports for this plugin.
func SetABISchemaHash(hash [abiSchemaHashLen]byte) {
	abiSchemaHash = hash
	abiSchemaHashSet = true
}

// SetABICapabilities sets the plugin capability bitmask exposed during handshake.
func SetABICapabilities(capabilities uint64) {
	abiCapabilities = capabilities
}

// SetABIMethods publishes the plugin's implemented method IDs.
func SetABIMethods(methodIDs []uint32) {
	if len(methodIDs) == 0 {
		abiMethods = nil
		abiMethodsRaw = nil
		return
	}
	abiMethods = append(abiMethods[:0], methodIDs...)
	slices.Sort(abiMethods)
	abiMethods = slices.Compact(abiMethods)
	abiMethodsRaw = make([]byte, len(abiMethods)*4)
	for i, methodID := range abiMethods {
		binary.LittleEndian.PutUint32(abiMethodsRaw[i*4:], methodID)
	}
}

//go:export __plugin_call
func pluginCall(methodID uint32, payloadSize uint32) bool {
	payload := ensureScratch(&pluginPayloadScratch, payloadSize)
	if ok := pluginRequest(bytesToPointer(payload)); !ok {
		message := "failed to load request payload from host"
		pluginError(stringToPointer(message), uint32(len(message)))
		return false
	}

	if dispatch := methodDispatcher; dispatch != nil {
		response, err := dispatch(methodID, payload)
		if err != nil {
			message := err.Error()
			pluginError(stringToPointer(message), uint32(len(message)))
			return false
		}
		pluginResponse(bytesToPointer(response), uint32(len(response)))
		return true
	}

	message := fmt.Sprintf("could not find method id %d", methodID)
	pluginError(stringToPointer(message), uint32(len(message)))
	return false
}

// Log is a convenience function to log messages to the console.
func Log(message string) {
	if len(message) == 0 {
		return
	}
	consoleLog(stringToPointer(message), uint32(len(message)))
}

// HostCallMethod invokes a host callback by numeric method ID.
func HostCallMethod(methodID uint32, payload []byte) ([]byte, error) {
	var out []byte
	err := HostCallMethodWithResponse(methodID, payload, func(response []byte) error {
		out = append([]byte(nil), response...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HostCallMethodWithResponse invokes a host callback by method ID and passes
// the response bytes to handle without allocating a result slice when the
// caller can decode immediately.
func HostCallMethodWithResponse(methodID uint32, payload []byte, handle func([]byte) error) error {
	result := hostCall(methodID, bytesToPointer(payload), uint32(len(payload)))
	if !result {
		errorLen := hostErrorLen()
		if errorLen == 0 {
			return &HostError{message: "host call failed without an error message"}
		}
		message := ensureScratch(&hostErrScratch, errorLen)
		hostError(bytesToPointer(message))

		return &HostError{message: string(message)}
	}

	responseLen := hostResponseLen()
	response := ensureScratch(&hostRespScratch, responseLen)
	hostResponse(bytesToPointer(response))
	if handle != nil {
		return handle(response)
	}
	return nil
}

//go:export __hookr_abi_version
func hookrABIVersion() uint32 {
	return uint32(abiMajor)<<16 | uint32(abiMinor)
}

//go:export __hookr_schema_hash
func hookrSchemaHash() uint64 {
	if !abiSchemaHashSet {
		return 0
	}
	return packPtrLenU64(uint32(bytesToPointer(abiSchemaHash[:])), abiSchemaHashLen)
}

//go:export __hookr_capabilities
func hookrCapabilities() uint64 {
	return abiCapabilities
}

//go:export __hookr_methods
func hookrMethods() uint64 {
	if len(abiMethodsRaw) == 0 {
		return 0
	}
	return packPtrLenU64(uint32(bytesToPointer(abiMethodsRaw)), uint32(len(abiMethodsRaw)))
}

//go:inline
func bytesToPointer(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

//go:inline
func stringToPointer(s string) uintptr {
	b := []byte(s)
	return bytesToPointer(b)
}

//go:inline
func packPtrLenU64(ptr uint32, dataLen uint32) uint64 {
	return uint64(ptr)<<32 | uint64(dataLen)
}

func ensureScratch(dst *[]byte, size uint32) []byte {
	if int(size) > cap(*dst) {
		*dst = make([]byte, int(size))
	}
	return (*dst)[:int(size)]
}

func (e *HostError) Error() string {
	return "Host error: " + e.message
}
