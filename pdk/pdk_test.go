//go:build !wasip1

package pdk

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestSetABIHelpersAndExports(t *testing.T) {
	SetABISchemaHash([abiSchemaHashLen]byte{1, 2, 3})
	SetABICapabilities(99)
	SetABIMethods([]uint32{7, 3, 7})

	if got := hookrABIVersion(); got != uint32(abiMajor)<<16|uint32(abiMinor) {
		t.Fatalf("unexpected abi version: %d", got)
	}
	if hookrCapabilities() != 99 {
		t.Fatalf("unexpected capabilities: %d", hookrCapabilities())
	}
	packed := hookrMethods()
	if packed == 0 {
		t.Fatal("expected packed methods")
	}
	methodLen := uint32(packed & 0xffffffff)
	if methodLen != 8 {
		t.Fatalf("unexpected method len: %d", methodLen)
	}
	if got := binary.LittleEndian.Uint32(abiMethodsRaw[0:4]); got != 3 {
		t.Fatalf("unexpected first method: %d", got)
	}
	if got := binary.LittleEndian.Uint32(abiMethodsRaw[4:8]); got != 7 {
		t.Fatalf("unexpected second method: %d", got)
	}
}

func TestSetABIMethodsEmptyClearsState(t *testing.T) {
	SetABIMethods([]uint32{1, 2})
	SetABIMethods(nil)
	if abiMethods != nil || abiMethodsRaw != nil || hookrMethods() != 0 {
		t.Fatalf("expected methods cleared")
	}
}

func TestPluginCallWithoutDispatcher(t *testing.T) {
	methodDispatcher = nil
	if pluginCall(42, 0) {
		t.Fatal("expected plugin call to fail without host payload support")
	}
}

func TestLogAndHostError(t *testing.T) {
	Log("")
	hostResponse(0)
	hostError(0)
	consoleLog(0, 0)
	if got := hostResponseLen(); got != 0 {
		t.Fatalf("unexpected host response len: %d", got)
	}
	if got := hostErrorLen(); got != 0 {
		t.Fatalf("unexpected host error len: %d", got)
	}
	err := (&HostError{message: "boom"}).Error()
	if !strings.Contains(err, "boom") {
		t.Fatalf("unexpected host error: %q", err)
	}
}

func TestHostCallMethodErrorPath(t *testing.T) {
	_, err := HostCallMethod(1, []byte("hello"))
	if err == nil {
		t.Fatal("expected host call error")
	}
	if !strings.Contains(err.Error(), "Host error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHelpers(t *testing.T) {
	buf := []byte("abc")
	if bytesToPointer(nil) != 0 {
		t.Fatal("nil bytes should map to zero")
	}
	if bytesToPointer(buf) == 0 {
		t.Fatal("non-empty bytes should map to pointer")
	}
	if stringToPointer("") != 0 {
		t.Fatal("empty string should map to zero")
	}
	if got := packPtrLenU64(2, 3); got != uint64(2)<<32|3 {
		t.Fatalf("unexpected packed ptr len: %d", got)
	}
	var scratch []byte
	if got := len(ensureScratch(&scratch, 5)); got != 5 {
		t.Fatalf("unexpected scratch len: %d", got)
	}
	if cap(scratch) < 5 {
		t.Fatal("expected grown scratch")
	}
}
