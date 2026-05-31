package flatc

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct {
	Path string
}

type GoOptions struct {
	SchemaPath   string
	OutDir       string
	PackageName  string
	ObjectAPI    bool
	IncludePaths []string
}

type RustOptions struct {
	SchemaPath   string
	OutDir       string
	IncludePaths []string
}

type CppOptions struct {
	SchemaPath   string
	OutDir       string
	IncludePaths []string
}

type SwiftOptions struct {
	SchemaPath   string
	OutDir       string
	IncludePaths []string
}

type TypeScriptOptions struct {
	SchemaPath   string
	OutDir       string
	IncludePaths []string
}

func New(path string) (*Runner, error) {
	if path == "" {
		path = "flatc"
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return nil, fmt.Errorf("find flatc: %w", err)
	}
	return &Runner{Path: resolved}, nil
}

func (r *Runner) GenerateBFBS(schemaPath, outDir string, includePaths []string) (string, error) {
	args := buildBFBSArgs(schemaPath, outDir, includePaths)
	if _, err := r.run(args...); err != nil {
		return "", err
	}
	name := filepath.Base(schemaPath)
	name = strings.TrimSuffix(name, filepath.Ext(name)) + ".bfbs"
	return filepath.Join(outDir, name), nil
}

func (r *Runner) GenerateGo(opts GoOptions) error {
	args := buildGoArgs(opts)
	_, err := r.run(args...)
	return err
}

func (r *Runner) GenerateRust(opts RustOptions) error {
	args := buildRustArgs(opts)
	_, err := r.run(args...)
	return err
}

func (r *Runner) GenerateCpp(opts CppOptions) error {
	args := buildCppArgs(opts)
	_, err := r.run(args...)
	return err
}

func (r *Runner) GenerateSwift(opts SwiftOptions) error {
	args := buildSwiftArgs(opts)
	_, err := r.run(args...)
	return err
}

func (r *Runner) GenerateTypeScript(opts TypeScriptOptions) error {
	args := buildTypeScriptArgs(opts)
	_, err := r.run(args...)
	return err
}

func (r *Runner) EncodeJSON(
	schemaPath string,
	includePaths []string,
	rootType string,
	rawJSON []byte,
) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "hookr-flatc-encode-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	wrapperPath, err := writeRootSchema(tmpDir, schemaPath, rootType)
	if err != nil {
		return nil, err
	}
	inputPath := filepath.Join(tmpDir, "input.json")
	if err := os.WriteFile(inputPath, rawJSON, 0o600); err != nil {
		return nil, fmt.Errorf("write input json: %w", err)
	}

	args := []string{"--binary", "--raw-binary", "--strict-json"}
	args = appendIncludeArgs(args, withSchemaDir(includePaths, schemaPath))
	args = append(args, "-o", tmpDir, wrapperPath, inputPath)
	if _, err := r.run(args...); err != nil {
		return nil, err
	}

	outPath := filepath.Join(tmpDir, "input.bin")
	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read encoded binary: %w", err)
	}
	return data, nil
}

func (r *Runner) DecodeJSON(
	schemaPath string,
	includePaths []string,
	rootType string,
	rawBinary []byte,
) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "hookr-flatc-decode-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	wrapperPath, err := writeRootSchema(tmpDir, schemaPath, rootType)
	if err != nil {
		return nil, err
	}
	inputPath := filepath.Join(tmpDir, "input.bin")
	if err := os.WriteFile(inputPath, rawBinary, 0o600); err != nil {
		return nil, fmt.Errorf("write input binary: %w", err)
	}

	args := []string{"--json", "--raw-binary", "--strict-json", "--defaults-json"}
	args = appendIncludeArgs(args, withSchemaDir(includePaths, schemaPath))
	args = append(args, "-o", tmpDir, wrapperPath, "--", inputPath)
	if _, err := r.run(args...); err != nil {
		return nil, err
	}

	outPath := filepath.Join(tmpDir, "input.json")
	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read decoded json: %w", err)
	}
	return data, nil
}

func buildBFBSArgs(schemaPath, outDir string, includePaths []string) []string {
	args := []string{"--binary", "--schema"}
	args = appendIncludeArgs(args, includePaths)
	args = append(args, "-o", outDir, schemaPath)
	return args
}

func buildGoArgs(opts GoOptions) []string {
	args := []string{"--go", "--go-namespace", opts.PackageName}
	if opts.ObjectAPI {
		args = append(args, "--gen-object-api")
	}
	args = appendIncludeArgs(args, opts.IncludePaths)
	args = append(args, "-o", opts.OutDir, opts.SchemaPath)
	return args
}

func buildRustArgs(opts RustOptions) []string {
	args := []string{"--rust"}
	args = appendIncludeArgs(args, opts.IncludePaths)
	args = append(args, "-o", opts.OutDir, opts.SchemaPath)
	return args
}

func buildCppArgs(opts CppOptions) []string {
	args := []string{"--cpp"}
	args = appendIncludeArgs(args, opts.IncludePaths)
	args = append(args, "-o", opts.OutDir, opts.SchemaPath)
	return args
}

func buildSwiftArgs(opts SwiftOptions) []string {
	args := []string{"--swift"}
	args = appendIncludeArgs(args, opts.IncludePaths)
	args = append(args, "-o", opts.OutDir, opts.SchemaPath)
	return args
}

func buildTypeScriptArgs(opts TypeScriptOptions) []string {
	args := []string{"--ts"}
	args = appendIncludeArgs(args, opts.IncludePaths)
	args = append(args, "-o", opts.OutDir, opts.SchemaPath)
	return args
}

func appendIncludeArgs(args []string, includePaths []string) []string {
	for _, includePath := range includePaths {
		includePath = strings.TrimSpace(includePath)
		if includePath == "" {
			continue
		}
		args = append(args, "-I", includePath)
	}
	return args
}

func withSchemaDir(includePaths []string, schemaPath string) []string {
	schemaDir := filepath.Dir(schemaPath)
	if schemaDir == "" || schemaDir == "." {
		return includePaths
	}
	out := make([]string, 0, len(includePaths)+1)
	out = append(out, schemaDir)
	out = append(out, includePaths...)
	return out
}

func writeRootSchema(dir, schemaPath, rootType string) (string, error) {
	if rootType == "" {
		return "", errors.New("root type is required")
	}
	name := filepath.Base(schemaPath)
	wrapperPath := filepath.Join(dir, "root.fbs")
	content := fmt.Sprintf("include %q;\nroot_type %s;\n", name, rootType)
	if err := os.WriteFile(wrapperPath, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("write wrapper schema: %w", err)
	}
	return wrapperPath, nil
}

func (r *Runner) run(args ...string) ([]byte, error) {
	cmd := exec.Command(r.Path, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run %s %v: %w\n%s", r.Path, args, err, out.String())
	}
	return out.Bytes(), nil
}
