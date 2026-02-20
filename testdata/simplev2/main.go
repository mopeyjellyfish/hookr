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
	hookr.SetABIV2SchemaHash([32]byte{
		'h', 'o', 'o', 'k', 'r', '-', 'v', '2', '-', 'g', 'r', 'e', 'e', 't', 'e', 'r',
		'-', 's', 'c', 'h', 'e', 'm', 'a', '-', 'h', 'a', 's', 'h', '-', '0', '0', '1',
	})

	hookr.FnMethod(MethodEcho, EchoMethod)
	hookr.FnMethod(MethodVowel, VowelsMethod)
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
