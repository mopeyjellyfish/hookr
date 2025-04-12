package host

import "github.com/mopeyjellyfish/hookr/pkg/pdk/memory"

//export printString
func printString(uint64) uint64

func HostPrintString(data string) string {
	ptr := memory.WriteString(data)
	outPut := printString(ptr)
	return memory.ReadString(outPut)
}
