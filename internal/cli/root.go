package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mopeyjellyfish/hookr/internal/bench"
	"github.com/mopeyjellyfish/hookr/internal/buildkit"
	"github.com/mopeyjellyfish/hookr/internal/call"
	"github.com/mopeyjellyfish/hookr/internal/codegen"
	"github.com/mopeyjellyfish/hookr/internal/inspect"
	"github.com/mopeyjellyfish/hookr/internal/tui"
	"github.com/spf13/cobra"
)

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hookr",
		Short:         "Hookr CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newGenCommand(),
		newBuildCommand(),
		newBenchCommand(),
		newInspectCommand(),
		newCallCommand(),
		newTUICommand(),
	)

	return cmd
}

func newGenCommand() *cobra.Command {
	cfg := codegen.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate Hookr contract metadata and glue files from a schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := codegen.Generate(cfg); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(
				cmd.ErrOrStderr(),
				"hookr: generated package %s in %s\n",
				cfg.PackageName,
				filepath.Join(cfg.OutDir, cfg.PackageName),
			)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&cfg.SchemaPath, "schema", cfg.SchemaPath, "path to schema file")
	flags.StringVar(&cfg.OutDir, "out", cfg.OutDir, "output directory")
	flags.StringVar(&cfg.PackageName, "package", cfg.PackageName, "generated Go package name")
	flags.StringVar(&cfg.ContractName, "name", cfg.ContractName, "contract name override")
	flags.StringVar(&cfg.Lang, "lang", cfg.Lang, "target language to generate")
	flags.StringVar(&cfg.FlatcPath, "flatc", cfg.FlatcPath, "path to flatc binary")
	flags.StringSliceVarP(
		&cfg.IncludePaths,
		"include",
		"I",
		cfg.IncludePaths,
		"additional include directory for imported schemas (repeatable)",
	)
	flags.StringVar(
		&cfg.PluginService,
		"plugin-service",
		cfg.PluginService,
		"FlatBuffers plugin service name",
	)
	flags.StringVar(
		&cfg.OptionalAttribute,
		"optional-attribute",
		cfg.OptionalAttribute,
		"FlatBuffers attribute name for optional plugin methods",
	)
	flags.Uint64Var(
		&cfg.Capabilities,
		"capabilities",
		cfg.Capabilities,
		"ABI feature capability bitmask for generated contracts",
	)

	_ = cmd.MarkFlagRequired("schema")
	_ = cmd.MarkFlagRequired("out")
	_ = cmd.MarkFlagRequired("package")

	return cmd
}

func newNotImplementedCommand(use string, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New(use + " is not implemented yet")
		},
	}
}

func newBuildCommand() *cobra.Command {
	cfg := buildkit.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a Hookr plugin into a Wasm artifact",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Stderr = cmd.ErrOrStderr()
			return buildkit.Build(cfg)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(
		&cfg.PluginPath,
		"plugin",
		cfg.PluginPath,
		"path to the plugin package or main.go file",
	)
	flags.StringVar(&cfg.OutputPath, "out", cfg.OutputPath, "path to the output wasm file")
	flags.StringVar(&cfg.TinyGoPath, "tinygo", cfg.TinyGoPath, "path to tinygo binary")
	flags.StringVar(&cfg.Target, "target", cfg.Target, "tinygo target")
	flags.StringVar(&cfg.BuildMode, "buildmode", cfg.BuildMode, "tinygo build mode")
	flags.StringVar(&cfg.Scheduler, "scheduler", cfg.Scheduler, "tinygo scheduler")
	flags.BoolVar(&cfg.NoDebug, "no-debug", cfg.NoDebug, "disable debug info in tinygo build")

	_ = cmd.MarkFlagRequired("plugin")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}

func newInspectCommand() *cobra.Command {
	cfg := inspect.Config{}

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect Hookr contracts and Wasm modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Stdout = cmd.OutOrStdout()
			cfg.Stderr = cmd.ErrOrStderr()
			return inspect.Run(cfg)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&cfg.SchemaPath, "schema", cfg.SchemaPath, "path to the FlatBuffers schema")
	flags.StringVar(&cfg.PluginPath, "plugin", cfg.PluginPath, "path to the plugin artifact (.wasm)")
	flags.StringVar(
		&cfg.HostFixturePath,
		"host-fixture",
		cfg.HostFixturePath,
		"path to a host callback fixture file",
	)
	flags.StringVar(&cfg.Hash, "hash", cfg.Hash, "expected SHA-256 hash for the plugin wasm")
	flags.BoolVar(
		&cfg.AllowUnsigned,
		"allow-unsigned",
		cfg.AllowUnsigned,
		"allow loading an unsigned plugin wasm",
	)
	flags.StringVar(&cfg.FlatcPath, "flatc", cfg.FlatcPath, "path to flatc binary")
	flags.StringSliceVarP(
		&cfg.IncludePaths,
		"include",
		"I",
		cfg.IncludePaths,
		"additional include directory for imported schemas (repeatable)",
	)
	flags.StringVar(&cfg.Package, "package", cfg.Package, "Go package name used for generation")
	flags.StringVar(
		&cfg.PluginService,
		"plugin-service",
		codegen.DefaultConfig().PluginService,
		"FlatBuffers plugin service name",
	)
	flags.StringVar(
		&cfg.OptionalAttribute,
		"optional-attribute",
		codegen.DefaultConfig().OptionalAttribute,
		"FlatBuffers attribute name for optional plugin methods",
	)

	_ = cmd.MarkFlagRequired("schema")

	return cmd
}

func newCallCommand() *cobra.Command {
	cfg := call.Config{}

	cmd := &cobra.Command{
		Use:   "call",
		Short: "Invoke a plugin method from a schema and wasm module",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Stdin = cmd.InOrStdin()
			cfg.Stdout = cmd.OutOrStdout()
			cfg.Stderr = cmd.ErrOrStderr()
			return call.Run(cfg)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&cfg.SchemaPath, "schema", cfg.SchemaPath, "path to the FlatBuffers schema")
	flags.StringVar(&cfg.PluginPath, "plugin", cfg.PluginPath, "path to the plugin artifact (.wasm)")
	flags.StringVar(&cfg.Method, "method", cfg.Method, "plugin method name to invoke")
	flags.StringVar(
		&cfg.InputPath,
		"input",
		cfg.InputPath,
		"path to request JSON file (defaults to stdin)",
	)
	flags.StringVar(
		&cfg.HostFixturePath,
		"host-fixture",
		cfg.HostFixturePath,
		"path to a host callback fixture file",
	)
	flags.StringVar(&cfg.Hash, "hash", cfg.Hash, "expected SHA-256 hash for the plugin wasm")
	flags.BoolVar(
		&cfg.AllowUnsigned,
		"allow-unsigned",
		cfg.AllowUnsigned,
		"allow loading an unsigned plugin wasm",
	)
	flags.StringVar(&cfg.FlatcPath, "flatc", cfg.FlatcPath, "path to flatc binary")
	flags.StringSliceVarP(
		&cfg.IncludePaths,
		"include",
		"I",
		cfg.IncludePaths,
		"additional include directory for imported schemas (repeatable)",
	)
	flags.StringVar(&cfg.Package, "package", cfg.Package, "Go package name used for generation")
	flags.StringVar(
		&cfg.PluginService,
		"plugin-service",
		codegen.DefaultConfig().PluginService,
		"FlatBuffers plugin service name",
	)
	flags.StringVar(
		&cfg.OptionalAttribute,
		"optional-attribute",
		codegen.DefaultConfig().OptionalAttribute,
		"FlatBuffers attribute name for optional plugin methods",
	)

	_ = cmd.MarkFlagRequired("schema")
	_ = cmd.MarkFlagRequired("plugin")
	_ = cmd.MarkFlagRequired("method")

	return cmd
}

func newTUICommand() *cobra.Command {
	cfg := tui.Config{}

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Open an interactive terminal UI for a Hookr plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Stdin = cmd.InOrStdin()
			cfg.Stdout = cmd.OutOrStdout()
			return tui.Run(cfg)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&cfg.SchemaPath, "schema", cfg.SchemaPath, "path to the FlatBuffers schema")
	flags.StringVar(&cfg.PluginPath, "plugin", cfg.PluginPath, "path to the plugin artifact (.wasm)")
	flags.StringVar(
		&cfg.HostFixturePath,
		"host-fixture",
		cfg.HostFixturePath,
		"path to a host callback fixture file",
	)
	flags.StringVar(&cfg.Hash, "hash", cfg.Hash, "expected SHA-256 hash for the plugin wasm")
	flags.BoolVar(
		&cfg.AllowUnsigned,
		"allow-unsigned",
		cfg.AllowUnsigned,
		"allow loading an unsigned plugin wasm",
	)
	flags.StringVar(&cfg.FlatcPath, "flatc", cfg.FlatcPath, "path to flatc binary")
	flags.StringSliceVarP(
		&cfg.IncludePaths,
		"include",
		"I",
		cfg.IncludePaths,
		"additional include directory for imported schemas (repeatable)",
	)
	flags.StringVar(&cfg.Package, "package", cfg.Package, "Go package name used for generation")
	flags.StringVar(
		&cfg.PluginService,
		"plugin-service",
		codegen.DefaultConfig().PluginService,
		"FlatBuffers plugin service name",
	)
	flags.StringVar(
		&cfg.OptionalAttribute,
		"optional-attribute",
		codegen.DefaultConfig().OptionalAttribute,
		"FlatBuffers attribute name for optional plugin methods",
	)

	_ = cmd.MarkFlagRequired("schema")
	_ = cmd.MarkFlagRequired("plugin")

	return cmd
}

func newBenchCommand() *cobra.Command {
	cfg := bench.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "bench",
		Short: "Run Hookr benchmark scenarios",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Stdout = cmd.OutOrStdout()
			cfg.Stderr = cmd.ErrOrStderr()
			return bench.Run(cfg)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(
		&cfg.PackagePath,
		"package",
		cfg.PackagePath,
		"Go package path containing benchmarks",
	)
	flags.StringVar(&cfg.Bench, "bench", cfg.Bench, "benchmark regex passed to go test")
	flags.StringVar(&cfg.Run, "run", cfg.Run, "test regex passed to go test")
	flags.IntVar(&cfg.Count, "count", cfg.Count, "benchmark run count")

	return cmd
}
