//go:build !wasip1

package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	urlbalancerhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/urlbalancer/gen/urlbalancerhookr"
)

type host struct{}

func (host) RngInt(_ context.Context, req *urlbalancerhookr.RngIntRequestT) (*urlbalancerhookr.RngIntResponseT, error) {
	midpoint := req.Min
	if req.Max > req.Min {
		midpoint = req.Min + ((req.Max - req.Min) / 2)
	}
	return &urlbalancerhookr.RngIntResponseT{Value: midpoint}, nil
}

func (host) RngFloat(_ context.Context, _ *urlbalancerhookr.RngFloatRequestT) (*urlbalancerhookr.RngFloatResponseT, error) {
	return &urlbalancerhookr.RngFloatResponseT{Value: 0.5}, nil
}

func main() {
	ctx := context.Background()
	wasmPath := filepath.Join("testdata", "contracts", "urlbalancer", "bin", "urlbalancer.wasm")

	rt, err := urlbalancerhookr.Open(ctx, urlbalancerhookr.Config{
		WasmPath:    wasmPath,
		FileOptions: []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()},
		Host:        host{},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := rt.Close(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	info, err := rt.GetInfo(ctx, &urlbalancerhookr.EmptyT{})
	if err != nil {
		log.Fatal(err)
	}
	result, err := rt.Balance(ctx, &urlbalancerhookr.BalanceRequestT{
		Url:   "https://example.com/api",
		Nodes: []string{"node-a", "node-b", "node-c"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s %s -> %s\n", info.Name, info.Version, result.SelectedNode)
}
