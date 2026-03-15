//go:build wasip1

package main

import (
	tickloophookr "github.com/mopeyjellyfish/hookr/testdata/contracts/tickloop/gen/tickloophookr"
)

type pluginNoWarmup struct{}

func (pluginNoWarmup) GetInfo(_ *tickloophookr.PluginContext, _ *tickloophookr.EmptyT) (*tickloophookr.PluginInfoT, error) {
	return &tickloophookr.PluginInfoT{
		Name:        "tickloop-no-warmup",
		Version:     "1.0.0",
		Description: "Tick loop plugin without optional warmup method.",
	}, nil
}

func (pluginNoWarmup) Tick(ctx *tickloophookr.PluginContext, req *tickloophookr.TickRequestT) (*tickloophookr.TickResponseT, error) {
	if req == nil {
		req = &tickloophookr.TickRequestT{}
	}
	rng, err := ctx.Rng.Int(&tickloophookr.RngIntRequestT{Min: -1, Max: 1})
	if err != nil {
		return nil, err
	}
	return &tickloophookr.TickResponseT{
		NextStateHash: req.StateHash + uint64(req.Events),
		ContinueRun:   true,
		Actions:       req.Events,
		JitterBucket:  rng.Value,
		Note:          "tick processed without warmup",
	}, nil
}

//go:wasmexport hookr_init
func hookrInit() {
	tickloophookr.MustRegisterPlugin(pluginNoWarmup{})
}

func main() {}
