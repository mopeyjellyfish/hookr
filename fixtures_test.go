package hookr

import (
	"os/exec"
	"testing"
)

func TestFixtures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fixture e2e tests in short mode")
	}

	packages := []string{
		"./testdata/contracts/urlbalancer",
		"./testdata/contracts/textfilter",
		"./testdata/contracts/textfilter/gen/textfilterhookr",
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
