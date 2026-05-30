package codegen

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/mopeyjellyfish/hookr/internal/contract"
)

type Config struct {
	SchemaPath        string
	OutDir            string
	PackageName       string
	ContractName      string
	Lang              string
	FlatcPath         string
	IncludePaths      []string
	PluginService     string
	OptionalAttribute string
	Capabilities      uint64
}

func (c Config) Validate() error {
	if c.SchemaPath == "" || c.OutDir == "" || c.PackageName == "" {
		return errors.New("schema, out, and package are required")
	}
	if !isSupportedLang(c.Lang) {
		return errors.New("hookr gen currently supports only --lang go or --lang rust")
	}
	if !strings.HasSuffix(strings.ToLower(c.SchemaPath), ".fbs") {
		return errors.New("hookr gen requires a FlatBuffers schema (*.fbs)")
	}
	return nil
}

func DefaultConfig() Config {
	return Config{
		Lang:              "go",
		PluginService:     contract.DefaultPluginService,
		OptionalAttribute: contract.OptionalAttribute,
	}
}

func Usage(binaryName string) string {
	return fmt.Sprintf(
		"usage: %s -schema <file.fbs> -out <dir> -package <name> [-name contract]",
		binaryName,
	)
}

func ParseFlags(binaryName string, args []string) (Config, error) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet(binaryName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.SchemaPath, "schema", "", "path to FlatBuffers schema file (required)")
	fs.StringVar(&cfg.OutDir, "out", "", "output directory (required)")
	fs.StringVar(&cfg.PackageName, "package", "", "generated package or crate module name (required)")
	fs.StringVar(&cfg.ContractName, "name", "", "contract name override (optional)")
	fs.StringVar(&cfg.Lang, "lang", cfg.Lang, "target language to generate (go or rust)")
	fs.StringVar(&cfg.FlatcPath, "flatc", cfg.FlatcPath, "path to flatc binary")
	includeFlag := &stringSliceFlag{dst: &cfg.IncludePaths}
	fs.Var(
		includeFlag,
		"include",
		"additional include directory for imported schemas (repeatable or comma-separated)",
	)
	fs.Var(includeFlag, "I", "shorthand for --include")
	fs.StringVar(
		&cfg.PluginService,
		"plugin-service",
		cfg.PluginService,
		"FlatBuffers plugin service name",
	)
	fs.StringVar(
		&cfg.OptionalAttribute,
		"optional-attribute",
		cfg.OptionalAttribute,
		"FlatBuffers attribute name used for optional plugin methods",
	)
	fs.Uint64Var(
		&cfg.Capabilities,
		"capabilities",
		cfg.Capabilities,
		"ABI feature capability bitmask to publish and require for this contract",
	)
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type stringSliceFlag struct {
	dst *[]string
}

func (s *stringSliceFlag) String() string {
	if s == nil || s.dst == nil {
		return ""
	}
	return strings.Join(*s.dst, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	if s == nil || s.dst == nil {
		return nil
	}
	for _, raw := range strings.Split(value, ",") {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		*s.dst = append(*s.dst, item)
	}
	return nil
}

func RunCLI(binaryName string, args []string, stderr io.Writer) int {
	cfg, err := ParseFlags(binaryName, args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, Usage(binaryName))
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := Generate(cfg); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func Generate(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	return generateFlatBuffers(cfg)
}

func isSupportedLang(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go", "rust":
		return true
	default:
		return false
	}
}

func toExportedIdentifier(s string) string {
	if s == "" {
		return "Unnamed"
	}
	var b strings.Builder
	upperNext := true
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if b.Len() == 0 && unicode.IsDigit(r) {
				b.WriteByte('M')
			}
			if upperNext {
				r = unicode.ToUpper(r)
			}
			b.WriteRune(r)
			upperNext = false
		default:
			upperNext = true
		}
	}
	if b.Len() == 0 {
		return "Unnamed"
	}
	return b.String()
}
