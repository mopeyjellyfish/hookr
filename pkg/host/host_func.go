package host

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

type WasmFunc func(input uint64) uint64
type GoFunc func(ctx context.Context, input []byte) ([]byte, error)

type HostFunction struct {
	Func GoFunc
	Name string
}

func (h *HostFunction) Call(ctx context.Context, mem Memory, inPtr uint64) uint64 {
	var input []byte = nil
	var err error
	if inPtr > 0 {
		input, err = mem.Read(inPtr)
		if err != nil {
			fmt.Printf("error reading memory function (%s): %v\n", h.Name, err)
			return 0
		}
	}
	output, err := h.Func(ctx, input)
	if err != nil {
		fmt.Printf("error calling host function (%s): %v\n", h.Name, err)
		return 0
	}
	if len(output) == 0 {
		return 0
	}
	ptr, err := mem.Write(output)
	if err != nil {
		fmt.Printf("error writing memory function (%s): %v\n", h.Name, err)
		return 0
	}
	return ptr
}

func NewHostFunction(f GoFunc, name string, params []api.ValueType, returns []api.ValueType) HostFunction {
	return HostFunction{
		Func: f,
		Name: name,
	}
}
