package testutil

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const MinTinyGoVersion = "0.41.0"

var tinyGoCheck struct {
	once       sync.Once
	skipReason string
}

func RequireTinyGo(t testing.TB) {
	t.Helper()

	tinyGoCheck.once.Do(func() {
		tinygo, err := exec.LookPath("tinygo")
		if err != nil {
			tinyGoCheck.skipReason = "tinygo not installed"
			return
		}
		output, err := exec.Command(tinygo, "version").CombinedOutput()
		versionText := strings.TrimSpace(string(output))
		if err != nil {
			tinyGoCheck.skipReason = fmt.Sprintf("tinygo version failed: %v: %s", err, versionText)
			return
		}
		version, ok := parseTinyGoVersion(versionText)
		if !ok {
			tinyGoCheck.skipReason = fmt.Sprintf("could not parse tinygo version from %q", versionText)
			return
		}
		if compareVersions(version, mustParseVersion(MinTinyGoVersion)) < 0 {
			tinyGoCheck.skipReason = fmt.Sprintf(
				"tinygo %s installed; plugin build tests require tinygo %s or newer",
				formatVersion(version),
				MinTinyGoVersion,
			)
		}
	})

	if tinyGoCheck.skipReason != "" {
		t.Skip(tinyGoCheck.skipReason)
	}
}

type version struct {
	major int
	minor int
	patch int
}

var tinyGoVersionPattern = regexp.MustCompile(`\btinygo version ([0-9]+)\.([0-9]+)\.([0-9]+)\b`)

func parseTinyGoVersion(output string) (version, bool) {
	matches := tinyGoVersionPattern.FindStringSubmatch(output)
	if matches == nil {
		return version{}, false
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	return version{major: major, minor: minor, patch: patch}, true
}

func mustParseVersion(s string) version {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		panic("invalid version: " + s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		panic("invalid version: " + s)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		panic("invalid version: " + s)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		panic("invalid version: " + s)
	}
	return version{major: major, minor: minor, patch: patch}
}

func compareVersions(left, right version) int {
	if left.major != right.major {
		return left.major - right.major
	}
	if left.minor != right.minor {
		return left.minor - right.minor
	}
	return left.patch - right.patch
}

func formatVersion(v version) string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}
