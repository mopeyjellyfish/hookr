//go:build !wasip1

package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	textfilterhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/textfilter/gen/textfilterhookr"
)

func main() {
	ctx := context.Background()
	wasmPath := filepath.Join("testdata", "contracts", "textfilter", "bin", "textfilter.wasm")

	rt, err := textfilterhookr.Open(ctx, textfilterhookr.Config{
		WasmPath:    wasmPath,
		FileOptions: []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if closeErr := rt.Close(ctx); closeErr != nil {
			log.Fatal(closeErr)
		}
	}()

	info, err := rt.GetInfo(ctx, &textfilterhookr.EmptyT{})
	if err != nil {
		log.Fatal(err)
	}
	resp, err := rt.Filter(ctx, &textfilterhookr.FilterRequestT{
		Input:           "this platform has bad words and worse habits",
		BlockedTerms:    []string{"bad", "worse"},
		Replacement:     "[filtered]",
		CaseSensitive:   false,
		MaxReplacements: 3,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s %s => %s\n", info.Name, info.Version, resp.Output)
}
