//go:build wasip1

package main

import (
	tickloophookr "github.com/mopeyjellyfish/hookr/testdata/contracts/tickloop/gen/tickloophookr"
)

type plugin struct{}

func (plugin) GetInfo(_ *tickloophookr.PluginContext, _ *tickloophookr.EmptyT) (*tickloophookr.PluginInfoT, error) {
	return &tickloophookr.PluginInfoT{
		Name:        "tickloop",
		Version:     "1.0.0",
		Description: "Tick loop plugin with optional warmup support.",
	}, nil
}

func (plugin) Tick(ctx *tickloophookr.PluginContext, req *tickloophookr.TickRequestT) (*tickloophookr.TickResponseT, error) {
	if req == nil {
		req = &tickloophookr.TickRequestT{}
	}
	rng, err := ctx.Rng.Int(&tickloophookr.RngIntRequestT{Min: -4, Max: 4})
	if err != nil {
		return nil, err
	}
	jitter := rng.Value
	actions := req.Events + uint32(abs32(jitter))
	nextHash := req.StateHash + uint64(req.Events) + uint64(uint32(abs32(jitter)))
	return &tickloophookr.TickResponseT{
		NextStateHash: nextHash,
		ContinueRun:   req.Tick < 1000,
		Actions:       actions,
		JitterBucket:  jitter,
		Note:          "tick processed",
	}, nil
}

func (plugin) Warmup(_ *tickloophookr.PluginContext, req *tickloophookr.WarmupRequestT) (*tickloophookr.WarmupResponseT, error) {
	if req == nil {
		req = &tickloophookr.WarmupRequestT{}
	}
	ready := req.TargetTicks >= 32
	recommended := uint32(16)
	if req.TargetTicks > 0 {
		recommended = req.TargetTicks / 2
		if recommended == 0 {
			recommended = 1
		}
	}
	return &tickloophookr.WarmupResponseT{
		Ready:            ready,
		RecommendedBatch: recommended,
	}, nil
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

//go:wasmexport hookr_init
func hookrInit() {
	tickloophookr.MustRegisterPlugin(plugin{})
}

func main() {}
