package version

import (
	"runtime/debug"
	"strings"
)

var (
	// Value is optionally overridden at build time via -ldflags for packaged builds.
	Value = ""
)

type Info struct {
	Version  string
	Revision string
	Modified bool
}

func Current() string {
	return Read().Version
}

func Read() Info {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fallbackInfo()
	}

	if Value != "" {
		return Info{
			Version:  Value,
			Revision: readSetting(info, "vcs.revision"),
			Modified: readSetting(info, "vcs.modified") == "true",
		}
	}

	mainVersion := info.Main.Version
	if mainVersion != "" && mainVersion != "(devel)" {
		return Info{
			Version:  mainVersion,
			Revision: readSetting(info, "vcs.revision"),
			Modified: readSetting(info, "vcs.modified") == "true",
		}
	}

	revision := readSetting(info, "vcs.revision")
	modified := readSetting(info, "vcs.modified") == "true"
	if revision != "" {
		return Info{
			Version:  formatDevelVersion(revision, modified),
			Revision: revision,
			Modified: modified,
		}
	}

	return fallbackInfo()
}

func fallbackInfo() Info {
	if Value != "" {
		return Info{Version: Value}
	}
	return Info{Version: "dev"}
}

func readSetting(info *debug.BuildInfo, key string) string {
	for _, setting := range info.Settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return ""
}

func formatDevelVersion(revision string, modified bool) string {
	short := revision
	if len(short) > 12 {
		short = short[:12]
	}

	var b strings.Builder
	b.WriteString("devel-")
	b.WriteString(short)
	if modified {
		b.WriteString("-dirty")
	}
	return b.String()
}
