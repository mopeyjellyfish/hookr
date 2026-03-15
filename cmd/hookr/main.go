package main

import (
	"fmt"
	"os"

	"github.com/mopeyjellyfish/hookr/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
