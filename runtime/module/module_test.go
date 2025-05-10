package module

import (
	"context"
	"testing"
)

func TestHookrModuleErrors(t *testing.T) {
	m := hookrModule{}
	results := make([]uint64, 4)
	results[0] = 1
	results[1] = 2
	results[2] = 3
	results[3] = 4
	m.hostCall(context.Background(), nil, results)
	m.hostErrorLen(context.Background(), results)
	m.hostError(context.Background(), nil, results)
	m.hostResponseLen(context.Background(), results)
	m.hostResponse(context.Background(), nil, results)
	m.pluginRequest(context.Background(), nil, results)
	m.pluginResponse(context.Background(), nil, results)
	m.pluginError(context.Background(), nil, results)
}
