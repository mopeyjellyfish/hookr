package host

import (
	"testing"

	"github.com/mopeyjellyfish/hookr/file"
	"github.com/stretchr/testify/assert"
)

func TestInvalidFilePath(t *testing.T) {
	file, err := NewFile("invalid.wasm")
	assert.Nil(t, file)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error reading path: invalid.wasm")
}

func TestNoPathProvided(t *testing.T) {
	file, err := NewFile("")
	assert.Nil(t, file)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestValidNewFile(t *testing.T) {
	filePath := file.Find(simple, nil)
	wasmFile, err := NewFile(filePath)
	assert.Nil(t, err)
	assert.NotNil(t, wasmFile)
	assert.Equal(t, "main", wasmFile.Name)
	assert.Equal(t, filePath, wasmFile.Path)
}

func TestSettingOpts(t *testing.T) {
	filePath := file.Find(simple, nil)
	wasmFile, err := NewFile(filePath, Name("test"))
	assert.Nil(t, err)
	assert.NotNil(t, wasmFile)
	assert.Equal(t, "test", wasmFile.Name)
	assert.Equal(t, filePath, wasmFile.Path)

	wasmFile, err = NewFile(filePath, Hash("test"))
	assert.Nil(t, err)
	assert.NotNil(t, wasmFile)
	assert.Equal(t, "main", wasmFile.Name)
	assert.Equal(t, "test", wasmFile.Hash)
	assert.Equal(t, filePath, wasmFile.Path)

	wasmFile, err = NewFile(filePath, Name("test"), Hash("test"))
	assert.Nil(t, err)
	assert.NotNil(t, wasmFile)
	assert.Equal(t, "test", wasmFile.Name)
	assert.Equal(t, "test", wasmFile.Hash)
	assert.Equal(t, filePath, wasmFile.Path)

	wasmFile, err = NewFile(filePath, Name(""), Hash("test"))
	assert.Nil(t, err)
	assert.NotNil(t, wasmFile)
	assert.Equal(t, "main", wasmFile.Name)
	assert.Equal(t, "test", wasmFile.Hash)
	assert.Equal(t, filePath, wasmFile.Path)

	wasmFile, err = NewFile(filePath, Name(""), Hash(""))
	assert.Nil(t, err)
	assert.NotNil(t, wasmFile)
	assert.Equal(t, "main", wasmFile.Name)
	assert.Equal(t, "", wasmFile.Hash)
	assert.Equal(t, filePath, wasmFile.Path)
}

func TestGetWasmData(t *testing.T) {
	filePath := file.Find(simple, nil)
	wasmFile, err := NewFile(filePath)
	assert.Nil(t, err)
	data, err := wasmFile.GetData()
	assert.Nil(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, "main", data.Name)
	assert.Equal(t, filePath, wasmFile.Path)
	wasmData, err := data.GetData()
	assert.Nil(t, err)
	assert.NotNil(t, wasmData)
}
