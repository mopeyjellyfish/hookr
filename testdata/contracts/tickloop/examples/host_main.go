//go:build !wasip1

package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	tickloophookr "github.com/mopeyjellyfish/hookr/testdata/contracts/tickloop/gen/tickloophookr"
)

type host struct{}

func (host) Int(_ context.Context, req *tickloophookr.RngIntRequestT) (*tickloophookr.RngIntResponseT, error) {
	mid := req.Min
	if req.Max > req.Min {
		mid = req.Min + ((req.Max - req.Min) / 2)
	}
	return &tickloophookr.RngIntResponseT{Value: mid}, nil
}

func main() {
	ctx := context.Background()
	wasmPath := filepath.Join("testdata", "contracts", "tickloop", "bin", "tickloop.wasm")

	rt, err := tickloophookr.Open(ctx, tickloophookr.Config{
		PluginPath:  wasmPath,
		FileOptions: []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()},
		Host: tickloophookr.Host{
			Rng: host{},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := rt.Close(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	resp, err := rt.Tick(ctx, &tickloophookr.TickRequestT{
		Tick:      1,
		DtMicros:  16666,
		StateHash: 100,
		Events:    2,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("next=%d actions=%d jitter=%d\n", resp.NextStateHash, resp.Actions, resp.JitterBucket)
}
