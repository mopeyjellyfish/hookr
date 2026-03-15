package trustopts

import (
	"errors"

	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
)

// Build returns file options matching the runtime trust model.
// Tooling must either pin a hash or explicitly opt into unsigned artifacts.
func Build(hash string, allowUnsigned bool) ([]hookrruntime.FileOption, error) {
	if hash != "" {
		return []hookrruntime.FileOption{hookrruntime.WithHash(hash)}, nil
	}
	if allowUnsigned {
		return []hookrruntime.FileOption{hookrruntime.WithAllowUnsigned()}, nil
	}
	return nil, errors.New("plugin trust not configured: provide a hash or set allow unsigned")
}
