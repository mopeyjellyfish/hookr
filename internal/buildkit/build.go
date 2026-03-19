package buildkit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Config struct {
	PluginPath string
	OutputPath string
	GoPath     string
	BuildMode  string
	Stderr     io.Writer
}

func DefaultConfig() Config {
	return Config{
		BuildMode: "c-shared",
	}
}

func (c Config) Validate() error {
	if c.PluginPath == "" {
		return errors.New("plugin path is required")
	}
	if c.OutputPath == "" {
		return errors.New("output path is required")
	}
	return nil
}

func Build(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	errOut := writerOrDefault(cfg.Stderr, os.Stderr)
	goBin := cfg.GoPath
	if goBin == "" {
		goBin = "go"
	}
	resolved, err := exec.LookPath(goBin)
	if err != nil {
		return fmt.Errorf("find go: %w", err)
	}
	args := []string{
		"build",
		"-o", cfg.OutputPath,
	}
	if cfg.BuildMode != "" {
		args = append(args, "-buildmode="+cfg.BuildMode)
	}
	args = append(args, cfg.PluginPath)
	_, _ = fmt.Fprintf(errOut, "hookr: building plugin %s -> %s\n", cfg.PluginPath, cfg.OutputPath)

	cmd := exec.Command(resolved, args...)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s %v: %w\n%s", resolved, args, err, out.String())
	}
	_, _ = fmt.Fprintf(errOut, "hookr: built plugin %s\n", cfg.OutputPath)
	return nil
}

func writerOrDefault(w io.Writer, fallback io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return fallback
}
