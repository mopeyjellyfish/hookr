package host

import (
	"testing"

	"github.com/mopeyjellyfish/hookr/file"
	"github.com/stretchr/testify/assert"
)

func TestSpecInvalidInput(t *testing.T) {
	spec, err := NewSpecFromFiles(nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no wasm files provided")
	assert.Nil(t, spec)
}

func TestSpecInvalidPathInput(t *testing.T) {
	spec, err := NewSpecFromFiles([]string{"invalid.wasm"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error creating spec:")
	assert.Nil(t, spec)
}

func TestSpecValidInput(t *testing.T) {
	filePath := file.Find(simple, nil)
	spec, err := NewSpecFromFiles([]string{filePath})
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	assert.Len(t, spec.Wasm, 1)
}

func TestSpecsSetWasi(t *testing.T) {
	filePath := file.Find(simple, nil)
	spec, err := NewSpecFromFiles([]string{filePath}, Wasi(false))
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	assert.False(t, spec.Wasi)
}

func TestSpecWithFunction(t *testing.T) {
	filePath := file.Find(simple, nil)
	spec, err := NewSpecFromFiles([]string{filePath}, WithHostFunc(
		HostFunction{
			Name: "printString",
			Func: nil,
		},
	))
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	assert.Equal(t, 1, len(spec.HostFuncs))
}

func TestSpecWithFile(t *testing.T) {
	filePath := file.Find(simple, nil)
	spec, err := NewSpecFromFile(filePath)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	assert.Len(t, spec.Wasm, 1)
}

func TestAddHostFunc(t *testing.T) {
	spec := &Spec{}
	assert.Equal(t, 0, len(spec.HostFuncs))
	spec.AddHostFunc(HostFunction{})
	assert.Equal(t, 1, len(spec.HostFuncs))
}
