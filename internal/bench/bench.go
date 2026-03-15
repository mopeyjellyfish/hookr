package bench

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

type Config struct {
	PackagePath string
	Bench       string
	Run         string
	Count       int
	Stdout      io.Writer
	Stderr      io.Writer
}

func DefaultConfig() Config {
	return Config{
		PackagePath: "./testdata/contracts/tickloop",
		Bench:       ".",
		Run:         "^$",
		Count:       1,
	}
}

func Run(cfg Config) error {
	if cfg.PackagePath == "" {
		return errors.New("package path is required")
	}
	args := []string{
		"test",
		cfg.PackagePath,
		"-run", cfg.Run,
		"-bench", cfg.Bench,
		"-count", strconv.Itoa(cfg.Count),
	}
	cmd := exec.Command("go", args...)
	cmd.Stdout = writerOrDefault(cfg.Stdout, os.Stdout)
	cmd.Stderr = writerOrDefault(cfg.Stderr, os.Stderr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run go %v: %w", args, err)
	}
	return nil
}

func writerOrDefault(w io.Writer, fallback io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return fallback
}
