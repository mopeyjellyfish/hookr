package host

import (
	"github.com/mopeyjellyfish/hookr/pkg/pdk/memory"
)

//export randomNumber
func randomNumber(uint64) uint64

//export randomString
func randomString(uint64) uint64

type RandInput struct {
	Min   int64 `json:"min"`
	Max   int64 `json:"max"`
	Count int   `json:"count"`
}

type RandOutput struct {
	Numbers []int64 `json:"numbers"`
}

func RandomNumber(min int, max int, count int) RandOutput {
	input := RandInput{
		Min:   0,
		Max:   1000,
		Count: 10,
	}
	ptr := memory.WriteJson(input)
	outPut := randomNumber(ptr)
	randResult := RandOutput{}
	err := memory.ReadJson(outPut, &randResult)
	if err != nil {
		panic(err)
	}
	return randResult
}

type RandomStringInput struct {
	Length int    `json:"length"`
	Chars  string `json:"chars"`
}

type RandomOutput struct {
	String  string  `json:"string"`
	Numbers []int64 `json:"numbers"`
}

func RandomString(length int) RandomOutput {
	input := RandomStringInput{
		Length: length,
		Chars:  "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	}
	inPtr := memory.WriteJson(input)
	outPtr := randomString(inPtr)
	output := RandomOutput{}
	err := memory.ReadJson(outPtr, &output)
	if err != nil {
		panic(err)
	}
	return output
}
