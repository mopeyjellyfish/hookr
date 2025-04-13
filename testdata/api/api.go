package api

//go:generate msgp

type EchoRequest struct {
	Data string `msg:"data"`
}
type EchoResponse struct {
	Data string `msg:"data"`
}

type HelloRequest struct {
	Msg string `msg:"msg"`
}

type HelloResponse struct {
	Msg string `msg:"msg"`
}
