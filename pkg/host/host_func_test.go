package host

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateNewHostFunc(t *testing.T) {
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		return nil, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
}

func TestHostFuncCall(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		return nil, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	output := hostFn.Call(ctx, mem, 0)
	assert.Equal(t, uint64(0), output)
}

func TestReadMemoryError(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		output := []byte("output")
		return output, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	offset, err := mem.Write([]byte("input"))
	mem.SetReadError(errors.New("read error"))
	resultOffset := hostFn.Call(ctx, mem, offset)
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), offset)
	assert.Equal(t, uint64(0), resultOffset)
}

func TestWriteMemoryError(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		output := []byte("output")
		return output, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	offset, err := mem.Write([]byte("input"))
	mem.SetWriteError(errors.New("write error"))
	resultOffset := hostFn.Call(ctx, mem, offset)
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), offset)
	assert.Equal(t, uint64(0), resultOffset)
}

func TestFunctionErrorsWithInput(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		if string(input) == "error" {
			return nil, errors.New("test error")
		}
		if string(input) == "nooutput" {
			return nil, nil
		}
		return []byte("tester"), nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	inputOffset, err := mem.Write([]byte("error"))
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), inputOffset)
	output := hostFn.Call(ctx, mem, inputOffset)
	assert.Equal(t, uint64(0), output)

	newInputOffset, err := mem.Write([]byte("other"))
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), newInputOffset)
	output = hostFn.Call(ctx, mem, newInputOffset)
	assert.NotEqual(t, uint64(0), output)
	data, err := mem.Read(output)
	assert.Nil(t, err)
	assert.Equal(t, []byte("tester"), data)

	noOutputOffset, err := mem.Write([]byte("nooutput"))
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), noOutputOffset)
	output = hostFn.Call(ctx, mem, noOutputOffset)
	assert.Equal(t, uint64(0), output)
}

func TestFunctionReturnsData(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		output := []byte("output")
		return output, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	offset, err := mem.Write([]byte("input"))
	resultOffset := hostFn.Call(ctx, mem, offset)
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), offset)
	data, err := mem.Read(resultOffset)
	assert.Nil(t, err)
	assert.Equal(t, []byte("output"), data)
}

func TestMultipleWrites(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		output := []byte("output")
		return output, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	offset1, err := mem.Write([]byte("input"))
	assert.Nil(t, err)
	offset2, err := mem.Write([]byte("input1"))
	assert.Nil(t, err)
	offset3, err := mem.Write([]byte("input2"))
	resultOffset := hostFn.Call(ctx, mem, offset3)
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), offset3)
	assert.NotEqual(t, uint64(0), offset2)
	assert.NotEqual(t, uint64(0), offset1)
	data, err := mem.Read(resultOffset)
	assert.Nil(t, err)
	assert.Equal(t, []byte("output"), data)

}

func TestMallocErrors(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		output := []byte("output")
		return output, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	offset1, err := mem.Write([]byte("input"))
	assert.Nil(t, err)
	mem.SetMallocError(errors.New("malloc error"))
	resultOffset := hostFn.Call(ctx, mem, offset1)
	assert.Equal(t, uint64(0), resultOffset)
}

func TestMockMemory(t *testing.T) {
	mem := NewMockMemory()
	offset, err := mem.Malloc(10)
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), offset)
	assert.Equal(t, uint64(1), offset)
	data, err := mem.Read(offset)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(data))
	mem.Free(offset)
	assert.Nil(t, err)
	result, err := mem.Read(0)
	assert.Nil(t, result)
	assert.NotNil(t, err)
	var offsets []uint64
	for i := 0; i < 10; i++ {
		offset, err := mem.Write([]byte("test")) // write more values
		assert.Nil(t, err)
		assert.NotEqual(t, uint64(0), offset)
		offsets = append(offsets, offset)
	}
	for _, offset := range offsets {
		value, err := mem.Read(offset)
		assert.Nil(t, err)
		assert.Equal(t, []byte("test"), value)
		mem.Free(offset)
	}
	data, err = mem.Read(99999999) // Invalid ptr
	assert.NotNil(t, err)
	assert.Nil(t, data)
}

func TestFunctionCallNoParams(t *testing.T) {
	mem := NewMockMemory()
	ctx := context.Background()
	testFn := func(ctx context.Context, input []byte) ([]byte, error) {
		output := []byte("output")
		return output, nil
	}
	hostFn := NewHostFunction(testFn, "test", nil, nil)
	assert.NotNil(t, hostFn)
	resultOffset := hostFn.Call(ctx, mem, 0)
	data, err := mem.Read(resultOffset)
	assert.Nil(t, err)
	assert.Equal(t, []byte("output"), data)
}
