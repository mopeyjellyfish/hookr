package host

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/mopeyjellyfish/hookr/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type RandInput struct {
	Min   int64 `json:"min"`
	Max   int64 `json:"max"`
	Count int   `json:"count"`
}

type RandOutput struct {
	Numbers []int64 `json:"numbers"`
}

func getRandInt(min int64, max int64, count int) ([]int64, error) {
	rangeSize := big.NewInt(int64(max - min))
	numbers := make([]int64, count)
	for i := 0; i < count; i++ {
		n, err := rand.Int(rand.Reader, rangeSize)
		if err != nil {
			return nil, err
		}
		numbers[i] = n.Int64() + int64(min)
	}
	return numbers, nil
}

// Functions can define their own schemas for input and output, this one is Json
func randomInt(ctx context.Context, input []byte) ([]byte, error) {
	var parseInput RandInput
	err := json.Unmarshal(input, &parseInput)
	if err != nil {
		return nil, err
	}
	numbers, err := getRandInt(parseInput.Min, parseInput.Max, parseInput.Count)
	if err != nil {
		return nil, err
	}
	randResult := RandOutput{
		Numbers: numbers,
	}
	output, err := json.Marshal(&randResult)
	if err != nil {
		fmt.Printf("error marshalling output: %v\n", err)
		return nil, err
	}
	return output, nil
}

type RandomStringInput struct {
	Length int    `json:"length"`
	Chars  string `json:"chars"`
}

type RandomStringOutput struct {
	String  string  `json:"string"`
	Numbers []int64 `json:"numbers"`
}

// Functions can define their own schemas for input and output, this one is JSON
func randomString(ctx context.Context, input []byte) ([]byte, error) {
	var parseInput RandomStringInput
	err := json.Unmarshal(input, &parseInput)
	if err != nil {
		return nil, err
	}
	b := make([]byte, parseInput.Length)
	numbers, err := getRandInt(0, int64(len(parseInput.Chars)), parseInput.Length)
	if err != nil {
		return nil, err
	}
	for i := range b {
		b[i] = parseInput.Chars[numbers[i]]
	}
	output := RandomStringOutput{
		String:  string(b),
		Numbers: numbers,
	}
	data, err := json.Marshal(&output)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func getTestSpec(files []string) (*Spec, error) {
	if files == nil {
		files = []string{file.Find(simple, nil)}
	}
	goFunc := func(ctx context.Context, input []byte) ([]byte, error) {
		fmt.Printf("Host function called with input: %s\n", string(input))
		return []byte("host echo: " + string(input)), nil
	}
	spec, err := NewSpecFromFiles(files, WithHostFuncs(
		[]HostFunction{
			{
				Name: "randomNumber",
				Func: randomInt,
			},
			{
				Name: "printString", // What is imported in the wasm file
				Func: goFunc,        // The actual function that will run
			},
		},
	))
	spec.AddHostFunc(HostFunction{
		Name: "randomString",
		Func: randomString,
	})
	return spec, err
}

func TestPlugin(t *testing.T) {
	spec, err := getTestSpec(nil)
	require.Nil(t, err, "error creating spec: %v", err)
	require.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	require.Nil(t, err, "error creating plugin: %v", err)
	assert.NotNil(t, plugin)
	assert.True(t, plugin.Exists("blank"))
	assert.False(t, plugin.Exists("unknownfunction"))
}

func TestClashingPlugins(t *testing.T) {
	filePath := file.Find(simple, nil)
	spec, err := getTestSpec([]string{filePath, filePath})
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, plugin)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "module with name main already exists")
}

func TestUnknownFunction(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.False(t, plugin.Exists("unknownfunction"))
}

func TestPluginReadWrite(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.True(t, plugin.Exists("blank"))
	assert.False(t, plugin.Exists("unknownfunction"))
	data := []byte("test")
	ptr, err := plugin.WriteMem(data)
	assert.Nil(t, err)
	assert.NotEqual(t, uint64(0), ptr)
	output, err := plugin.ReadMem(ptr)
	assert.Nil(t, err)
	assert.Equal(t, data, output)
}

func TestPluginWriteNothing(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.True(t, plugin.Exists("blank"))
	assert.False(t, plugin.Exists("unknownfunction"))
	ptr, err := plugin.WriteMem([]byte{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no input provided")
	assert.Equal(t, uint64(0), ptr)
}

func TestPluginReadBadPtr(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.True(t, plugin.Exists("blank"))
	assert.False(t, plugin.Exists("unknownfunction"))
	output, err := plugin.ReadMem(0)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no pointer provided")
	assert.Nil(t, output)
}

func TestPluginCall(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)

	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	data := []byte("test")
	output, err := plugin.Call("echo", data)
	assert.Nil(t, err)
	assert.Equal(t, "echo: test", string(output))
	assert.Nil(t, err)
}

func TestPluginSerialize(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)

	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	data := []byte("test")
	output, err := plugin.Call("echojson", data)
	assert.Nil(t, err)
	assert.NotNil(t, output)
	fmt.Printf("Output: %s\n", string(output))
}

func TestCallUnknownPlugin(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.False(t, plugin.Exists("unknownfunction"))
	output, err := plugin.Call("unknownfunction", []byte{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "function unknownfunction does not exist")
	assert.Nil(t, output)

}

func TestReturnNoValue(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	output, err := plugin.Call("blank", []byte{})
	assert.Nil(t, err)
	assert.Nil(t, output)
}

func TestPluginMemory(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.True(t, plugin.Exists("blank"))
	assert.False(t, plugin.Exists("unknownfunction"))
	memory := plugin.Memory("memory")
	assert.NotNil(t, memory)
}

func TestPluginErrorReturn(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	output, err := plugin.Call("error", []byte{})
	assert.NotNil(t, err)
	assert.Nil(t, output)

}

func TestPluginNothingFunction(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	output, err := plugin.Call("nothing", []byte{})
	assert.Nil(t, err)
	assert.Nil(t, output)

}

func TestPluginError(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	assert.True(t, plugin.Exists("blank"))
	assert.False(t, plugin.Exists("unknownfunction"))
	output, err := plugin.GetError()
	assert.Nil(t, err)
	assert.Nil(t, output)
}

func TestPluginPanics(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	output, err := plugin.Call("panicFn", []byte{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error calling function panicFn")
	assert.Nil(t, output)
}

func TestPluginInvalidFormat(t *testing.T) {
	filePath := file.Find(invalidFormat, nil)
	spec, err := NewSpecFromFiles([]string{filePath})
	assert.Nil(t, err)
	assert.NotNil(t, spec)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error instantiating module")
	assert.Nil(t, plugin)
}

func TestPluginCallHostFunction(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	data := []byte("test")
	output, err := plugin.Call("hostEchoString", data)
	assert.Nil(t, err)
	assert.Equal(t, "host echo: test", string(output))
	assert.Nil(t, err)
}

type echoRand struct {
	RandomEcho []int64 `json:"randomEcho"`
}

func TestJsonInternalCallFunction(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	output, err := plugin.Call("echoRandNumber", nil)
	assert.Nil(t, err)
	assert.NotNil(t, output)
	var echo echoRand
	err = json.Unmarshal(output, &echo)
	assert.Nil(t, err)
	assert.NotNil(t, echo.RandomEcho)
	assert.Equal(t, 10, len(echo.RandomEcho))
}

func TestJsonHostFunctionCall(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	output, err := plugin.Call("getRandomString", nil)
	require.Nil(t, err)
	require.NotNil(t, output)
	var echo RandomStringOutput
	err = json.Unmarshal(output, &echo)
	assert.Nil(t, err)
	assert.NotNil(t, echo.String)
	assert.NotNil(t, echo.Numbers)
	assert.Equal(t, 10, len(echo.Numbers))
}

func TestPluginCallMultipleFunctions(t *testing.T) {
	spec, err := getTestSpec(nil)
	assert.Nil(t, err)
	ctx := context.Background()
	plugin, err := NewFromSpec(ctx, spec)
	assert.Nil(t, err)
	assert.NotNil(t, plugin)
	data := []byte("test")
	output, err := plugin.Call("hostEchoString", data)
	assert.Nil(t, err)
	assert.Equal(t, "host echo: test", string(output))
	assert.Nil(t, err)
	output, err = plugin.Call("echoRandNumber", nil)
	assert.Nil(t, err)
	assert.NotNil(t, output)
	var echo echoRand
	err = json.Unmarshal(output, &echo)
	assert.Nil(t, err)
	assert.NotNil(t, echo.RandomEcho)
	assert.Equal(t, 10, len(echo.RandomEcho))
}
