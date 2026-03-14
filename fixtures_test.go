package hookr

import (
	"os/exec"
	"testing"
)

func TestFixtures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fixture e2e tests in short mode")
	}
	if _, err := exec.LookPath("tinygo"); err != nil {
		t.Skip("tinygo not installed")
	}

	packages := []string{
		"./testdata/contracts/urlbalancer",
		"./testdata/contracts/textfilter",
		"./testdata/contracts/tickloop",
	}

	for _, pkg := range packages {
		pkg := pkg
		t.Run(pkg, func(t *testing.T) {
			cmd := exec.Command("go", "test", pkg, "-count=1")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go test %s failed: %v\n%s", pkg, err, output)
			}
		})
	}
}
