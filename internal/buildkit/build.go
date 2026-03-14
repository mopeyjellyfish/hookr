package buildkit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Config struct {
	PluginPath string
	OutputPath string
	TinyGoPath string
	Target     string
	BuildMode  string
	Scheduler  string
	NoDebug    bool
	Stderr     io.Writer
}

func DefaultConfig() Config {
	return Config{
		Target:    "wasip1",
		BuildMode: "c-shared",
		Scheduler: "none",
		NoDebug:   true,
	}
}

func (c Config) Validate() error {
	if c.PluginPath == "" {
		return fmt.Errorf("plugin path is required")
	}
	if c.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	return nil
}

func Build(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	errOut := writerOrDefault(cfg.Stderr, os.Stderr)
	tinygo := cfg.TinyGoPath
	if tinygo == "" {
		tinygo = "tinygo"
	}
	resolved, err := exec.LookPath(tinygo)
	if err != nil {
		return fmt.Errorf("find tinygo: %w", err)
	}
	args := []string{
		"build",
		"-o", cfg.OutputPath,
		"-target=" + cfg.Target,
		"-buildmode=" + cfg.BuildMode,
		"-scheduler=" + cfg.Scheduler,
	}
	if cfg.NoDebug {
		args = append(args, "--no-debug")
	}
	args = append(args, cfg.PluginPath)
	_, _ = fmt.Fprintf(errOut, "hookr: building plugin %s -> %s\n", cfg.PluginPath, cfg.OutputPath)

	cmd := exec.Command(resolved, args...)
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
