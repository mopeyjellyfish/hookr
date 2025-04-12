package file

import (
	"fmt"
	"os"
	"path/filepath"
)

// Find returns the absolute path of the given file in the Go module's
// It searches for the 'go.mod' file from the current working directory upwards
// and appends the envFile to the directory containing 'go.mod'.
// It panics if it fails to find the 'go.mod' file.
func Find(envFile string, getDirectory func() (string, error)) string {
	if getDirectory == nil {
		getDirectory = os.Getwd
	}
	currentDir, err := getDirectory()
	if err != nil {
		panic(err)
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			break
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			panic(fmt.Errorf("go.mod not found"))
		}
		currentDir = parent
	}

	return filepath.Join(currentDir, envFile)
}
