package version

import "runtime/debug"

var (
	// Value is overridden at build time via -ldflags.
	Value = "dev"
)

func Current() string {
	if Value != "" && Value != "dev" {
		return Value
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Value
	}
	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return Value
	}
	return info.Main.Version
}
