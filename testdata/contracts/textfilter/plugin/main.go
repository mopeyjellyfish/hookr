//go:build wasip1

package main

import (
	"strings"

	textfilterhookr "github.com/mopeyjellyfish/hookr/testdata/contracts/textfilter/gen/textfilterhookr"
)

type plugin struct{}

func (plugin) GetInfo(_ *textfilterhookr.PluginContext, _ *textfilterhookr.EmptyT) (*textfilterhookr.PluginInfoT, error) {
	return &textfilterhookr.PluginInfoT{
		Name:        "textfilter",
		Version:     "1.0.0",
		Description: "Simple term filter fixture for Hookr end-to-end testing.",
	}, nil
}

func (plugin) Filter(_ *textfilterhookr.PluginContext, req *textfilterhookr.FilterRequestT) (*textfilterhookr.FilterResponseT, error) {
	if req == nil {
		return &textfilterhookr.FilterResponseT{
			Blocked: true,
			Reason:  "request is required",
		}, nil
	}
	resp := &textfilterhookr.FilterResponseT{Output: req.Input}
	if len(req.BlockedTerms) == 0 {
		resp.Reason = "no blocked terms configured"
		return resp, nil
	}

	output := req.Input
	replacement := req.Replacement
	if replacement == "" {
		replacement = "[redacted]"
	}
	remaining := int(req.MaxReplacements)
	limit := remaining > 0
	total := 0

	for _, term := range req.BlockedTerms {
		if term == "" {
			continue
		}
		if limit && remaining <= 0 {
			break
		}
		var replaced int
		output, replaced = replaceTerm(output, term, replacement, req.CaseSensitive, remaining, limit)
		if replaced > 0 {
			total += replaced
			if limit {
				remaining -= replaced
			}
		}
	}

	resp.Output = output
	resp.Changed = total > 0
	resp.Replacements = uint32(total)
	if !resp.Changed {
		resp.Blocked = true
		resp.Reason = "no blocked terms found"
	}
	return resp, nil
}

func replaceTerm(input, term, replacement string, caseSensitive bool, remaining int, limit bool) (string, int) {
	if caseSensitive {
		source := input
		var b strings.Builder
		replaced := 0
		start := 0
		for {
			if limit && replaced >= remaining {
				break
			}
			idx := strings.Index(source[start:], term)
			if idx < 0 {
				break
			}
			idx += start
			b.WriteString(source[start:idx])
			b.WriteString(replacement)
			start = idx + len(term)
			replaced++
		}
		if replaced == 0 {
			return input, 0
		}
		b.WriteString(source[start:])
		return b.String(), replaced
	}

	source := input
	sourceLower := strings.ToLower(source)
	termLower := strings.ToLower(term)

	var b strings.Builder
	replaced := 0
	start := 0
	for {
		if limit && replaced >= remaining {
			break
		}
		idx := strings.Index(sourceLower[start:], termLower)
		if idx < 0 {
			break
		}
		idx += start
		b.WriteString(source[start:idx])
		b.WriteString(replacement)
		start = idx + len(term)
		replaced++
	}
	if replaced == 0 {
		return input, 0
	}
	b.WriteString(source[start:])
	return b.String(), replaced
}

//go:wasmexport hookr_init
func hookrInit() {
	textfilterhookr.MustRegisterPlugin(plugin{})
}

func main() {}
