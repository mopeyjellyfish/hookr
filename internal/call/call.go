package call

import (
	"context"
	"fmt"
	"io"
	"os"
)

type Config struct {
	SchemaPath        string
	WasmPath          string
	Method            string
	InputPath         string
	HostFixturePath   string
	FlatcPath         string
	IncludePaths      []string
	Package           string
	PluginService     string
	HostService       string
	OptionalAttribute string
	Stdin             io.Reader
	Stdout            io.Writer
	Stderr            io.Writer
}

func Run(cfg Config) error {
	if cfg.SchemaPath == "" {
		return fmt.Errorf("schema path is required")
	}
	if cfg.WasmPath == "" {
		return fmt.Errorf("wasm path is required")
	}
	if cfg.Method == "" {
		return fmt.Errorf("method is required")
	}
	errOut := cfg.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	session, err := NewSession(cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()
	method, ok := session.Method(cfg.Method)
	if !ok {
		return fmt.Errorf("plugin method %q not found in contract", cfg.Method)
	}
	_, _ = fmt.Fprintf(errOut, "hookr: invoking %s on %s\n", method.Name, cfg.WasmPath)
	if cfg.HostFixturePath != "" {
		_, _ = fmt.Fprintf(errOut, "hookr: loaded host fixture %s\n", cfg.HostFixturePath)
	}
	input, err := readInput(cfg)
	if err != nil {
		return err
	}
	result, err := session.InvokeJSON(context.Background(), method.Name, input)
	if err != nil {
		return err
	}

	out := cfg.Stdout
	if out == nil {
		out = os.Stdout
	}
	if _, err := out.Write(result.ResponseJSON); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	_, _ = fmt.Fprintf(errOut, "hookr: call succeeded for %s\n", method.Name)
	return nil
}

func readInput(cfg Config) ([]byte, error) {
	if cfg.InputPath != "" && cfg.InputPath != "-" {
		data, err := os.ReadFile(cfg.InputPath)
		if err != nil {
			return nil, fmt.Errorf("read input: %w", err)
		}
		return data, nil
	}
	in := cfg.Stdin
	if in == nil {
		in = os.Stdin
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	return data, nil
}
