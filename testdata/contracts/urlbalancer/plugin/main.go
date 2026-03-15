//go:build wasip1

package main

import (
	"net/url"

	urlbalancerhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/urlbalancer/gen/urlbalancerhookr"
)

type plugin struct{}

func (plugin) GetInfo(_ *urlbalancerhookr.PluginContext, _ *urlbalancerhookr.EmptyT) (*urlbalancerhookr.PluginInfoT, error) {
	return &urlbalancerhookr.PluginInfoT{
		Name:        "urlbalancer",
		Version:     "1.0.0",
		Description: "Selects a backend node for a URL using host-provided RNG callbacks.",
	}, nil
}

func (plugin) Balance(ctx *urlbalancerhookr.PluginContext, req *urlbalancerhookr.BalanceRequestT) (*urlbalancerhookr.BalanceResponseT, error) {
	resp := &urlbalancerhookr.BalanceResponseT{}
	if req == nil {
		resp.Error = "request is required"
		return resp, nil
	}
	parsed, err := url.Parse(req.Url)
	if err != nil {
		resp.Error = err.Error()
		return resp, nil
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		resp.Error = "url must include scheme and host"
		return resp, nil
	}
	if len(req.Nodes) == 0 {
		resp.Error = "at least one node is required"
		return resp, nil
	}

	rngIntResp, err := ctx.Rng.Int(&urlbalancerhookr.RngIntRequestT{
		Min: 0,
		Max: int32(len(req.Nodes) - 1),
	})
	if err != nil {
		return nil, err
	}
	rngFloatResp, err := ctx.Rng.Float(&urlbalancerhookr.RngFloatRequestT{})
	if err != nil {
		return nil, err
	}
	index := int(rngIntResp.Value)
	if index < 0 || index >= len(req.Nodes) {
		resp.Error = "host returned rng index outside node range"
		return resp, nil
	}

	resp.Valid = true
	resp.NormalizedUrl = parsed.String()
	resp.Scheme = parsed.Scheme
	resp.Host = parsed.Host
	resp.SelectedNode = req.Nodes[index]
	resp.SelectedIndex = uint32(index)
	resp.RngInt = rngIntResp.Value
	resp.RngFloat = rngFloatResp.Value
	return resp, nil
}

//go:wasmexport hookr_init
func hookrInit() {
	urlbalancerhookr.MustRegisterPlugin(plugin{})
}

func main() {}
