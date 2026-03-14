package main

import (
	"fmt"

	hookr "github.com/mopeyjellyfish/hookr/pdk"
)

const (
	MethodHello uint32 = 1
	MethodEcho  uint32 = 2
	MethodVowel uint32 = 3
)

//go:wasmexport hookr_init
func Initialize() {
	// Static schema hash used to exercise optional runtime handshake validation.
	hookr.SetABISchemaHash([32]byte{
		'h', 'o', 'o', 'k', 'r', '-', 's', 'i', 'm', 'p', 'l', 'e', '-', 'm', 'e', 't',
		'h', 'o', 'd', '-', 's', 'c', 'h', 'e', 'm', 'a', '-', '0', '0', '0', '1', '!',
	})
	hookr.SetABIMethods([]uint32{MethodEcho, MethodVowel})
	hookr.SetMethodDispatcher(func(methodID uint32, payload []byte) ([]byte, error) {
		switch methodID {
		case MethodEcho:
			return EchoMethod(payload)
		case MethodVowel:
			return VowelsMethod(payload)
		default:
			return nil, fmt.Errorf("unknown method id %d", methodID)
		}
	})
}

func EchoMethod(payload []byte) ([]byte, error) {
	resp, err := hookr.HostCallMethod(MethodHello, payload)
	if err != nil {
		hookr.Log(err.Error())
		return nil, err
	}
	return resp, nil
}

func VowelsMethod(payload []byte) ([]byte, error) {
	vowelCount := 0
	for _, b := range payload {
		if b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u' ||
			b == 'A' || b == 'E' || b == 'I' || b == 'O' || b == 'U' {
			vowelCount++
		}
	}
	return []byte(fmt.Sprintf("%d", vowelCount)), nil
}
