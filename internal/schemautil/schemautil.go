package schemautil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mopeyjellyfish/hookr/internal/contract"
	"github.com/mopeyjellyfish/hookr/internal/flatbuffers/reflection"
	"github.com/mopeyjellyfish/hookr/internal/flatc"
)

type Config struct {
	SchemaPath        string
	FlatcPath         string
	IncludePaths      []string
	Package           string
	PluginService     string
	HostService       string
	OptionalAttribute string
}

func Load(cfg Config) (*flatc.Runner, contract.Contract, error) {
	runner, model, _, err := LoadWithReflection(cfg)
	return runner, model, err
}

func LoadWithReflection(cfg Config) (*flatc.Runner, contract.Contract, *reflection.Schema, error) {
	if cfg.SchemaPath == "" {
		return nil, contract.Contract{}, nil, fmt.Errorf("schema path is required")
	}
	pkg := cfg.Package
	if pkg == "" {
		pkg = filepath.Base(cfg.SchemaPath)
		pkg = strings.TrimSuffix(pkg, filepath.Ext(pkg))
	}
	runner, err := flatc.New(cfg.FlatcPath)
	if err != nil {
		return nil, contract.Contract{}, nil, err
	}
	tmpDir, err := os.MkdirTemp("", "hookr-schema-*")
	if err != nil {
		return nil, contract.Contract{}, nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	bfbsPath, err := runner.GenerateBFBS(cfg.SchemaPath, tmpDir, cfg.IncludePaths)
	if err != nil {
		return nil, contract.Contract{}, nil, err
	}
	data, err := os.ReadFile(bfbsPath)
	if err != nil {
		return nil, contract.Contract{}, nil, fmt.Errorf("read bfbs: %w", err)
	}
	model, err := contract.Load(contract.LoadOptions{
		SchemaPath:        cfg.SchemaPath,
		BFBSPath:          bfbsPath,
		PackageName:       pkg,
		PluginServiceName: cfg.PluginService,
		HostServiceName:   cfg.HostService,
		OptionalAttribute: cfg.OptionalAttribute,
	})
	if err != nil {
		return nil, contract.Contract{}, nil, err
	}
	return runner, model, reflection.GetRootAsSchema(data, 0), nil
}
