package codegen

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/mopeyjellyfish/hookr/internal/contract"
	"github.com/mopeyjellyfish/hookr/internal/flatc"
)

type flatbuffersTemplateData struct {
	PackageName          string
	ContractName         string
	SchemaPath           string
	RustSchemaModule     string
	RustNamespacePath    string
	SchemaHashHex        string
	SchemaHashLiterals   []string
	ContractCapabilities uint64
	PluginMethods        []flatbuffersMethodTemplate
	HostModules          []flatbuffersModuleTemplate
	UniqueTypes          []flatbuffersTypeTemplate
	HasHostModules       bool
}

type flatbuffersModuleTemplate struct {
	ServiceName   string
	GoName        string
	RustFieldName string
	RustTypeName  string
	InterfaceName string
	ClientName    string
	Methods       []flatbuffersMethodTemplate
}

type flatbuffersMethodTemplate struct {
	ID                uint32
	ServiceName       string
	Name              string
	GoName            string
	RustName          string
	ConstName         string
	RustConstName     string
	RequestType       string
	ResponseType      string
	Optional          bool
	OptionalIfaceName string
}

type flatbuffersTypeTemplate struct {
	TypeName string
}

func generateFlatBuffers(cfg Config) error {
	switch strings.ToLower(strings.TrimSpace(cfg.Lang)) {
	case "go":
		return generateFlatBuffersGo(cfg)
	case "rust":
		return generateFlatBuffersRust(cfg)
	case "zig":
		return generateFlatBuffersZig(cfg)
	default:
		return errors.New("flatbuffers generation currently supports only --lang go, --lang rust, or --lang zig")
	}
}

func generateFlatBuffersGo(cfg Config) error {
	if !strings.EqualFold(cfg.Lang, "go") {
		return errors.New("flatbuffers go generation requires --lang go")
	}
	runner, err := flatc.New(cfg.FlatcPath)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "hookr-flatbuffers-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	bfbsPath, err := runner.GenerateBFBS(cfg.SchemaPath, tmpDir, cfg.IncludePaths)
	if err != nil {
		return err
	}
	if err := runner.GenerateGo(flatc.GoOptions{
		SchemaPath:   cfg.SchemaPath,
		OutDir:       cfg.OutDir,
		PackageName:  cfg.PackageName,
		ObjectAPI:    true,
		IncludePaths: cfg.IncludePaths,
	}); err != nil {
		return err
	}

	model, err := contract.Load(contract.LoadOptions{
		SchemaPath:        cfg.SchemaPath,
		BFBSPath:          bfbsPath,
		PackageName:       cfg.PackageName,
		ContractName:      cfg.ContractName,
		PluginServiceName: cfg.PluginService,
		OptionalAttribute: cfg.OptionalAttribute,
	})
	if err != nil {
		return err
	}
	if err := validateFlatBuffersGoIdentifiers(model); err != nil {
		return err
	}

	packageDir := filepath.Join(cfg.OutDir, cfg.PackageName)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		return fmt.Errorf("create package output dir: %w", err)
	}

	td := flatbuffersTemplateData{
		PackageName:          cfg.PackageName,
		ContractName:         model.Name,
		SchemaPath:           cfg.SchemaPath,
		SchemaHashHex:        hex.EncodeToString(model.SchemaHash[:]),
		SchemaHashLiterals:   byteLiterals(model.SchemaHash[:]),
		ContractCapabilities: cfg.Capabilities,
		PluginMethods:        buildTemplateMethods(model.PluginService.Methods),
		HostModules:          buildTemplateModules(model.HostServices),
		UniqueTypes:          collectTemplateTypes(model),
		HasHostModules:       len(model.HostServices) > 0,
	}

	writes := []struct {
		path string
		tpl  string
	}{
		{filepath.Join(packageDir, "contract_meta_gen.go"), flatbuffersMetaTemplate},
		{filepath.Join(packageDir, "host_sdk_gen.go"), flatbuffersHostTemplate},
		{filepath.Join(packageDir, "plugin_pdk_gen.go"), flatbuffersPluginTemplate},
	}
	for _, write := range writes {
		if err := renderFlatbuffersTemplate(write.path, write.tpl, td); err != nil {
			return fmt.Errorf("generate %s: %w", filepath.Base(write.path), err)
		}
	}
	return nil
}

func generateFlatBuffersRust(cfg Config) error {
	runner, err := flatc.New(cfg.FlatcPath)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "hookr-flatbuffers-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	bfbsPath, err := runner.GenerateBFBS(cfg.SchemaPath, tmpDir, cfg.IncludePaths)
	if err != nil {
		return err
	}
	model, err := contract.Load(contract.LoadOptions{
		SchemaPath:        cfg.SchemaPath,
		BFBSPath:          bfbsPath,
		PackageName:       cfg.PackageName,
		ContractName:      cfg.ContractName,
		PluginServiceName: cfg.PluginService,
		OptionalAttribute: cfg.OptionalAttribute,
	})
	if err != nil {
		return err
	}

	packageDir := filepath.Join(cfg.OutDir, cfg.PackageName)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		return fmt.Errorf("create rust output dir: %w", err)
	}
	if err := runner.GenerateRust(flatc.RustOptions{
		SchemaPath:   cfg.SchemaPath,
		OutDir:       packageDir,
		IncludePaths: cfg.IncludePaths,
	}); err != nil {
		return err
	}

	td := flatbuffersTemplateData{
		PackageName:          cfg.PackageName,
		ContractName:         model.Name,
		SchemaPath:           cfg.SchemaPath,
		RustSchemaModule:     rustGeneratedModuleName(cfg.SchemaPath),
		RustNamespacePath:    rustNamespacePath(model),
		SchemaHashHex:        hex.EncodeToString(model.SchemaHash[:]),
		SchemaHashLiterals:   byteLiterals(model.SchemaHash[:]),
		ContractCapabilities: cfg.Capabilities,
		PluginMethods:        buildTemplateMethods(model.PluginService.Methods),
		HostModules:          buildTemplateModules(model.HostServices),
		UniqueTypes:          collectTemplateTypes(model),
		HasHostModules:       len(model.HostServices) > 0,
	}
	writes := []struct {
		path string
		tpl  string
	}{
		{filepath.Join(packageDir, "lib.rs"), flatbuffersRustLibTemplate},
		{filepath.Join(packageDir, "hookr_plugin.rs"), flatbuffersRustPluginTemplate},
	}
	for _, write := range writes {
		if err := renderFlatbuffersTemplate(write.path, write.tpl, td); err != nil {
			return fmt.Errorf("generate %s: %w", filepath.Base(write.path), err)
		}
	}
	return nil
}

func generateFlatBuffersZig(cfg Config) error {
	runner, err := flatc.New(cfg.FlatcPath)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "hookr-flatbuffers-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	bfbsPath, err := runner.GenerateBFBS(cfg.SchemaPath, tmpDir, cfg.IncludePaths)
	if err != nil {
		return err
	}
	model, err := contract.Load(contract.LoadOptions{
		SchemaPath:        cfg.SchemaPath,
		BFBSPath:          bfbsPath,
		PackageName:       cfg.PackageName,
		ContractName:      cfg.ContractName,
		PluginServiceName: cfg.PluginService,
		OptionalAttribute: cfg.OptionalAttribute,
	})
	if err != nil {
		return err
	}

	packageDir := filepath.Join(cfg.OutDir, cfg.PackageName)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		return fmt.Errorf("create zig output dir: %w", err)
	}

	td := flatbuffersTemplateData{
		PackageName:          cfg.PackageName,
		ContractName:         model.Name,
		SchemaPath:           cfg.SchemaPath,
		SchemaHashHex:        hex.EncodeToString(model.SchemaHash[:]),
		SchemaHashLiterals:   byteLiterals(model.SchemaHash[:]),
		ContractCapabilities: cfg.Capabilities,
		PluginMethods:        buildTemplateMethods(model.PluginService.Methods),
		HostModules:          buildTemplateModules(model.HostServices),
		UniqueTypes:          collectTemplateTypes(model),
		HasHostModules:       len(model.HostServices) > 0,
	}
	if err := renderFlatbuffersTemplate(filepath.Join(packageDir, "hookr_plugin.zig"), flatbuffersZigPluginTemplate, td); err != nil {
		return fmt.Errorf("generate hookr_plugin.zig: %w", err)
	}
	return nil
}

func validateFlatBuffersGoIdentifiers(model contract.Contract) error {
	idOwners := map[string]string{}
	typeQualified := map[string]string{}
	typeOwners := map[string]string{}
	reserveIdentifier := func(identifier string, owner string) {
		idOwners[identifier] = owner
	}
	reserveType := func(typeName string, owner string) {
		typeQualified[typeName] = owner
		typeOwners[typeName] = owner
	}
	registerIdentifier := func(kind string, identifier string, owner string) error {
		if prior, ok := idOwners[identifier]; ok && prior != owner {
			return fmt.Errorf(
				"generated Go %s identifier collision for %q between %s and %s",
				kind,
				identifier,
				prior,
				owner,
			)
		}
		idOwners[identifier] = owner
		return nil
	}
	registerType := func(typeName string, qualifiedName string, owner string) error {
		if priorQualified, ok := typeQualified[typeName]; ok && priorQualified != qualifiedName {
			return fmt.Errorf(
				"generated Go type helper collision for %q between %s (%s) and %s (%s)",
				typeName,
				typeOwners[typeName],
				priorQualified,
				owner,
				qualifiedName,
			)
		}
		typeQualified[typeName] = qualifiedName
		typeOwners[typeName] = owner
		return nil
	}
	for identifier, owner := range map[string]string{
		"PluginSchema":         "generated package value",
		"SchemaHash":           "generated package value",
		"ContractCapabilities": "generated package value",
		"Runtime":              "generated runtime type",
		"Config":               "generated config type",
		"ReloadConfig":         "generated reload config type",
		"Host":                 "generated host aggregate type",
		"PluginContext":        "generated plugin context type",
		"Plugin":               "generated plugin interface",
		"Open":                 "generated package function",
		"RegisterPlugin":       "generated package function",
		"MustRegisterPlugin":   "generated package function",
	} {
		reserveIdentifier(identifier, owner)
	}
	for typeName, owner := range map[string]string{
		"Runtime":       "generated.Runtime",
		"Config":        "generated.Config",
		"ReloadConfig":  "generated.ReloadConfig",
		"Host":          "generated.Host",
		"Plugin":        "generated.Plugin",
		"PluginContext": "generated.PluginContext",
	} {
		reserveType(typeName, owner)
	}
	checkPluginMethods := func(methods []contract.Method) error {
		for _, method := range methods {
			owner := method.ServiceName + "." + method.Name
			goName := toExportedIdentifier(method.Name)
			if goName == "Close" {
				return fmt.Errorf("generated Go method identifier collision for %q between generated Runtime.Close and %s", goName, owner)
			}
			if err := registerIdentifier("method", goName, owner); err != nil {
				return err
			}
			if err := registerIdentifier("constant", "Method"+toExportedIdentifier(method.ServiceName)+goName, owner); err != nil {
				return err
			}
			if method.Optional {
				if err := registerIdentifier("optional interface", "OptionalPlugin"+goName, owner); err != nil {
					return err
				}
			}
			if err := registerType(method.RequestType, method.RequestQualified, owner+" request"); err != nil {
				return err
			}
			if err := registerType(method.ResponseType, method.ResponseQualified, owner+" response"); err != nil {
				return err
			}
		}
		return nil
	}
	checkHostMethods := func(service contract.Service) error {
		methodOwners := map[string]string{}
		for _, method := range service.Methods {
			owner := method.ServiceName + "." + method.Name
			goName := toExportedIdentifier(method.Name)
			if prior, ok := methodOwners[goName]; ok && prior != owner {
				return fmt.Errorf(
					"generated Go host method identifier collision for %q between %s and %s",
					goName,
					prior,
					owner,
				)
			}
			methodOwners[goName] = owner
			if err := registerIdentifier("constant", "Method"+toExportedIdentifier(method.ServiceName)+goName, owner); err != nil {
				return err
			}
			if err := registerType(method.RequestType, method.RequestQualified, owner+" request"); err != nil {
				return err
			}
			if err := registerType(method.ResponseType, method.ResponseQualified, owner+" response"); err != nil {
				return err
			}
		}
		return nil
	}
	if err := checkPluginMethods(model.PluginService.Methods); err != nil {
		return err
	}
	moduleOwners := map[string]string{}
	for _, service := range model.HostServices {
		moduleName := toExportedIdentifier(service.Name)
		if prior, ok := moduleOwners[moduleName]; ok && prior != service.Name {
			return fmt.Errorf(
				"generated Go host module identifier collision for %q between %s and %s",
				moduleName,
				prior,
				service.Name,
			)
		}
		moduleOwners[moduleName] = service.Name
		if err := registerIdentifier("host interface", moduleName+"Host", service.Name); err != nil {
			return err
		}
		if err := registerIdentifier("host client", moduleName+"Client", service.Name); err != nil {
			return err
		}
		if err := checkHostMethods(service); err != nil {
			return err
		}
	}
	return nil
}

func buildTemplateMethods(methods []contract.Method) []flatbuffersMethodTemplate {
	out := make([]flatbuffersMethodTemplate, 0, len(methods))
	for _, method := range methods {
		exported := toExportedIdentifier(method.Name)
		serviceExported := toExportedIdentifier(method.ServiceName)
		out = append(out, flatbuffersMethodTemplate{
			ID:                method.ID,
			ServiceName:       method.ServiceName,
			Name:              method.Name,
			GoName:            exported,
			RustName:          toRustIdentifier(method.Name),
			ConstName:         "Method" + serviceExported + exported,
			RustConstName:     "METHOD_" + toRustConstIdentifier(method.ServiceName+"_"+method.Name),
			RequestType:       method.RequestType,
			ResponseType:      method.ResponseType,
			Optional:          method.Optional,
			OptionalIfaceName: "OptionalPlugin" + exported,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func buildTemplateModules(services []contract.Service) []flatbuffersModuleTemplate {
	out := make([]flatbuffersModuleTemplate, 0, len(services))
	for _, service := range services {
		goName := toExportedIdentifier(service.Name)
		out = append(out, flatbuffersModuleTemplate{
			ServiceName:   service.Name,
			GoName:        goName,
			RustFieldName: toRustIdentifier(service.Name),
			RustTypeName:  goName,
			InterfaceName: goName + "Host",
			ClientName:    goName + "Client",
			Methods:       buildTemplateMethods(service.Methods),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServiceName < out[j].ServiceName })
	return out
}

func collectTemplateTypes(model contract.Contract) []flatbuffersTypeTemplate {
	seen := map[string]struct{}{}
	collect := func(methods []contract.Method) {
		for _, method := range methods {
			seen[method.RequestType] = struct{}{}
			seen[method.ResponseType] = struct{}{}
		}
	}
	collect(model.PluginService.Methods)
	for _, service := range model.HostServices {
		collect(service.Methods)
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]flatbuffersTypeTemplate, 0, len(names))
	for _, name := range names {
		out = append(out, flatbuffersTypeTemplate{TypeName: name})
	}
	return out
}

func byteLiterals(raw []byte) []string {
	out := make([]string, 0, len(raw))
	for _, b := range raw {
		out = append(out, fmt.Sprintf("0x%02x", b))
	}
	return out
}

func rustGeneratedModuleName(schemaPath string) string {
	name := strings.TrimSuffix(filepath.Base(schemaPath), filepath.Ext(schemaPath))
	return toRustIdentifier(name) + "_generated"
}

func rustNamespacePath(model contract.Contract) string {
	for _, method := range model.PluginService.Methods {
		return rustNamespaceFromQualified(method.RequestQualified)
	}
	for _, service := range model.HostServices {
		for _, method := range service.Methods {
			return rustNamespaceFromQualified(method.RequestQualified)
		}
	}
	return ""
}

func rustNamespaceFromQualified(qualified string) string {
	parts := strings.Split(qualified, ".")
	if len(parts) <= 1 {
		return ""
	}
	modules := make([]string, 0, len(parts)-1)
	for _, part := range parts[:len(parts)-1] {
		modules = append(modules, toRustIdentifier(part))
	}
	return strings.Join(modules, "::")
}

func toRustIdentifier(s string) string {
	identifier := toDelimitedIdentifier(s, '_', false)
	if identifier == "" {
		identifier = "unnamed"
	}
	if identifier[0] >= '0' && identifier[0] <= '9' {
		identifier = "m_" + identifier
	}
	if isRustKeyword(identifier) {
		return identifier + "_"
	}
	return identifier
}

func toRustConstIdentifier(s string) string {
	identifier := toDelimitedIdentifier(s, '_', true)
	if identifier == "" {
		return "UNNAMED"
	}
	if identifier[0] >= '0' && identifier[0] <= '9' {
		return "M_" + identifier
	}
	return identifier
}

func toDelimitedIdentifier(s string, delimiter rune, upper bool) string {
	var b strings.Builder
	prevWasDelimiter := true
	prevWasLowerOrDigit := false
	for _, r := range s {
		isLetter := unicode.IsLetter(r)
		isDigit := unicode.IsDigit(r)
		if !isLetter && !isDigit {
			if b.Len() > 0 && !prevWasDelimiter {
				b.WriteRune(delimiter)
				prevWasDelimiter = true
			}
			prevWasLowerOrDigit = false
			continue
		}
		if unicode.IsUpper(r) && prevWasLowerOrDigit && !prevWasDelimiter {
			b.WriteRune(delimiter)
		}
		original := r
		if upper {
			r = unicode.ToUpper(r)
		} else {
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
		prevWasDelimiter = false
		prevWasLowerOrDigit = unicode.IsLower(original) || unicode.IsDigit(original)
	}
	out := strings.Trim(string([]rune(b.String())), string(delimiter))
	for strings.Contains(out, string(delimiter)+string(delimiter)) {
		out = strings.ReplaceAll(out, string(delimiter)+string(delimiter), string(delimiter))
	}
	return out
}

func isRustKeyword(identifier string) bool {
	switch identifier {
	case "as", "break", "const", "continue", "crate", "else", "enum", "extern",
		"false", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod",
		"move", "mut", "pub", "ref", "return", "self", "Self", "static", "struct",
		"super", "trait", "true", "type", "unsafe", "use", "where", "while",
		"async", "await", "dyn":
		return true
	default:
		return false
	}
}

func renderFlatbuffersTemplate(path string, tpl string, data flatbuffersTemplateData) error {
	t, err := template.New(filepath.Base(path)).Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(tpl)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	return t.Execute(f, data)
}

const flatbuffersMetaTemplate = `// Code generated by hookr. DO NOT EDIT.

package {{ .PackageName }}

var SchemaHash = [32]byte{ {{ join .SchemaHashLiterals ", " }} }
const ContractCapabilities uint64 = {{ .ContractCapabilities }}

{{ range .PluginMethods -}}
const {{ .ConstName }} uint32 = {{ .ID }}
{{ end }}

{{ if .HasHostModules }}
{{ range .HostModules }}
{{ range .Methods -}}
const {{ .ConstName }} uint32 = {{ .ID }}
{{ end }}
{{ end }}
{{ end }}
`

const flatbuffersHostTemplate = `//go:build !wasip1

// Code generated by hookr. DO NOT EDIT.

package {{ .PackageName }}

import (
	"context"
	"errors"
	"fmt"
	{{- if .HasHostModules }}
	"reflect"
	{{- end }}
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
)

var PluginSchema = runtimecontract.Schema{
	Name:       "{{ .ContractName }}",
	SchemaHash: SchemaHash,
	Capabilities: ContractCapabilities,
	Methods: []runtimecontract.Method{
			{{- range .PluginMethods }}
			{
				ID:           runtimecontract.MethodID({{ .ConstName }}),
				Name:         "{{ .Name }}",
				RequestType:  "{{ .RequestType }}",
				ResponseType: "{{ .ResponseType }}",
				Optional:     {{ .Optional }},
			},
			{{- end }}
		},
	}

type Runtime struct {
	rt runtimeInvoker
}

type Config struct {
	PluginPath     string
	FileOptions    []hookrruntime.FileOption
	{{- if .HasHostModules }}
	Host           Host
	{{- end }}
	Reload         *ReloadConfig
	RuntimeOptions []hookrruntime.Option
}

type ReloadConfig struct {
	Debounce      time.Duration
	OnReload      func(ctx context.Context, next *Runtime, event hookrruntime.ReloadEvent) error
	OnReloadError func(ctx context.Context, err error)
}

type runtimeInvoker interface {
	InvokeMethod(ctx context.Context, methodID uint32, payload []byte) ([]byte, error)
	InvokeMethodWithResponse(ctx context.Context, methodID uint32, payload []byte, fn func([]byte) error) error
	HasPluginMethodID(methodID uint32) bool
	PluginHandshake() (runtimecontract.Handshake, bool)
	Close(ctx context.Context) error
}

{{ if .HasHostModules -}}
type Host struct {
	{{- range .HostModules }}
	{{ .GoName }} {{ .InterfaceName }}
	{{- end }}
}

{{ range .HostModules -}}
type {{ .InterfaceName }} interface {
	{{- range .Methods }}
	{{ .GoName }}(ctx context.Context, req *{{ .RequestType }}T) (*{{ .ResponseType }}T, error)
	{{- end }}
}
{{ end -}}
{{- end }}

func Open(ctx context.Context, cfg Config) (*Runtime, error) {
	if cfg.PluginPath == "" {
		return nil, errors.New("plugin path is required")
	}
	{{ if .HasHostModules -}}
	if err := validateHostModules(cfg.Host); err != nil {
		return nil, err
	}
	{{- end }}
	opts := []hookrruntime.Option{
		hookrruntime.WithFile(cfg.PluginPath, cfg.FileOptions...),
		hookrruntime.WithContractSchema(PluginSchema),
	}
	{{ if .HasHostModules -}}
	opts = append(opts, hookrruntime.WithHostMethodFns(bindHostMethods(cfg.Host)...))
	{{- end }}
	opts = append(opts, cfg.RuntimeOptions...)
	if cfg.Reload != nil {
		reload := hookrruntime.ReloadConfig{
			Debounce:      cfg.Reload.Debounce,
			OnReloadError: cfg.Reload.OnReloadError,
		}
		if cfg.Reload.OnReload != nil {
			reload.OnReload = func(
				ctx context.Context,
				next hookrruntime.Invoker,
				event hookrruntime.ReloadEvent,
			) error {
				return cfg.Reload.OnReload(ctx, &Runtime{rt: next}, event)
			}
		}
		rt, err := hookrruntime.NewLive(ctx, reload, opts...)
		if err != nil {
			return nil, err
		}
		return &Runtime{rt: rt}, nil
	}
	rt, err := hookrruntime.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &Runtime{rt: rt}, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil || r.rt == nil {
		return nil
	}
	return r.rt.Close(ctx)
}

{{ range .PluginMethods }}
func (r *Runtime) Supports{{ .GoName }}() bool {
	if r == nil || r.rt == nil {
		return false
	}
	return r.rt.HasPluginMethodID({{ .ConstName }})
}

{{ end -}}

{{ range .PluginMethods }}
func (r *Runtime) {{ .GoName }}View(ctx context.Context, req *{{ .RequestType }}T, fn func(*{{ .ResponseType }}) error) error {
	if fn == nil {
		return errors.New("response callback is required")
	}
	return withEncoded{{ .RequestType }}(req, func(payload []byte) error {
		return r.rt.InvokeMethodWithResponse(ctx, {{ .ConstName }}, payload, func(response []byte) error {
			out, err := decode{{ .ResponseType }}View(response)
			if err != nil {
				return err
			}
			return fn(out)
		})
	})
}

func (r *Runtime) {{ .GoName }}(ctx context.Context, req *{{ .RequestType }}T) (*{{ .ResponseType }}T, error) {
	var out *{{ .ResponseType }}T
	err := r.{{ .GoName }}View(ctx, req, func(response *{{ .ResponseType }}) error {
		out = response.UnPack()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

{{ end -}}

{{ if .HasHostModules -}}
func validateHostModules(host Host) error {
	{{- range .HostModules }}
	if isNilHostModule(host.{{ .GoName }}) {
		return errors.New("host module {{ .ServiceName }} is required")
	}
	{{- end }}
	return nil
}

func isNilHostModule(module any) bool {
	if module == nil {
		return true
	}
	value := reflect.ValueOf(module)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func bindHostMethods(host Host) []hookrruntime.HostMethod {
	methods := make([]hookrruntime.HostMethod, 0)
	{{- range .HostModules }}
	if !isNilHostModule(host.{{ .GoName }}) {
		methods = append(methods, bind{{ .GoName }}HostMethods(host.{{ .GoName }})...)
	}
	{{- end }}
	return methods
}

{{ range .HostModules -}}
func bind{{ .GoName }}HostMethods(host {{ .InterfaceName }}) []hookrruntime.HostMethod {
	return []hookrruntime.HostMethod{
		{{- range .Methods }}
		hookrruntime.HostFnMethod({{ .ConstName }}, func(ctx context.Context, payload []byte) ([]byte, error) {
			req, err := decode{{ .RequestType }}(payload)
			if err != nil {
				return nil, err
			}
			resp, err := host.{{ .GoName }}(ctx, req)
			if err != nil {
				return nil, err
			}
			return encode{{ .ResponseType }}(resp)
		}),
		{{- end }}
	}
}

{{ end -}}
{{ end -}}

type flatbuffersPacker interface {
	Pack(*flatbuffers.Builder) flatbuffers.UOffsetT
}

var builderPool = make(chan *flatbuffers.Builder, 32)

func acquireBuilder() *flatbuffers.Builder {
	select {
	case builder := <-builderPool:
		builder.Reset()
		return builder
	default:
		return flatbuffers.NewBuilder(128)
	}
}

func releaseBuilder(builder *flatbuffers.Builder) {
	if builder == nil {
		return
	}
	builder.Reset()
	select {
	case builderPool <- builder:
	default:
	}
}

func decodeFlatbuffer(typeName string, payload []byte, decode func([]byte) any) (_ any, err error) {
	if len(payload) < flatbuffers.SizeUint32 {
		return nil, fmt.Errorf("decode %s: invalid flatbuffer payload", typeName)
	}
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("decode %s: invalid flatbuffer payload", typeName)
		}
	}()
	return decode(payload), nil
}

{{ range .UniqueTypes }}
func encode{{ .TypeName }}(msg *{{ .TypeName }}T) ([]byte, error) {
	if msg == nil {
		msg = &{{ .TypeName }}T{}
	}
	builder := acquireBuilder()
	defer releaseBuilder(builder)
	offset := msg.Pack(builder)
	Finish{{ .TypeName }}Buffer(builder, offset)
	return append([]byte(nil), builder.FinishedBytes()...), nil
}

func withEncoded{{ .TypeName }}(msg *{{ .TypeName }}T, fn func([]byte) error) error {
	if msg == nil {
		msg = &{{ .TypeName }}T{}
	}
	builder := acquireBuilder()
	defer releaseBuilder(builder)
	offset := msg.Pack(builder)
	Finish{{ .TypeName }}Buffer(builder, offset)
	return fn(builder.FinishedBytes())
}

func decode{{ .TypeName }}(payload []byte) (*{{ .TypeName }}T, error) {
	msg, err := decode{{ .TypeName }}View(payload)
	if err != nil {
		return nil, err
	}
	return msg.UnPack(), nil
}

func decode{{ .TypeName }}View(payload []byte) (*{{ .TypeName }}, error) {
	msg, err := decodeFlatbuffer("{{ .TypeName }}", payload, func(raw []byte) any {
		return GetRootAs{{ .TypeName }}(raw, 0)
	})
	if err != nil {
		return nil, err
	}
	return msg.(*{{ .TypeName }}), nil
}

{{ end -}}
`

const flatbuffersPluginTemplate = `//go:build wasip1

// Code generated by hookr. DO NOT EDIT.

package {{ .PackageName }}

import (
	"errors"
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/mopeyjellyfish/hookr/pdk"
	pdkcontract "github.com/mopeyjellyfish/hookr/pdk/contract"
)

{{ if .HasHostModules -}}
type PluginContext struct {
	{{- range .HostModules }}
	{{ .GoName }} {{ .ClientName }}
	{{- end }}
}
{{ else -}}
type PluginContext struct{}
{{ end }}

type Plugin interface {
	{{- range .PluginMethods }}
	{{- if not .Optional }}
	{{ .GoName }}(ctx *PluginContext, req *{{ .RequestType }}T) (*{{ .ResponseType }}T, error)
	{{- end }}
	{{- end }}
}

{{ range .PluginMethods -}}
{{ if .Optional }}
type {{ .OptionalIfaceName }} interface {
	{{ .GoName }}(ctx *PluginContext, req *{{ .RequestType }}T) (*{{ .ResponseType }}T, error)
}

{{ end -}}
{{ end -}}

{{ if .HasHostModules -}}
{{ range .HostModules }}
type {{ .ClientName }} struct{}

{{ $clientName := .ClientName -}}
{{ range .Methods }}
func (ctx {{ $clientName }}) {{ .GoName }}View(req *{{ .RequestType }}T, fn func(*{{ .ResponseType }}) error) error {
	if fn == nil {
		return errors.New("response callback is required")
	}
	return withEncoded{{ .RequestType }}(req, func(payload []byte) error {
		return pdk.HostCallMethodWithResponse({{ .ConstName }}, payload, func(response []byte) error {
			out, err := decode{{ .ResponseType }}View(response)
			if err != nil {
				return err
			}
			return fn(out)
		})
	})
}

func (ctx {{ $clientName }}) {{ .GoName }}(req *{{ .RequestType }}T) (*{{ .ResponseType }}T, error) {
	var out *{{ .ResponseType }}T
	err := ctx.{{ .GoName }}View(req, func(response *{{ .ResponseType }}) error {
		out = response.UnPack()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

{{ end -}}
{{ end }}
{{ end }}

func MustRegisterPlugin(plugin Plugin) {
	if err := RegisterPlugin(plugin); err != nil {
		panic(err)
	}
}

func RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return errors.New("plugin implementation is nil")
	}
	pdkcontract.PublishContractHandshake(SchemaHash, ContractCapabilities, implementedPluginMethodIDs(plugin))
	pdk.SetMethodDispatcher(func(methodID uint32, payload []byte) ([]byte, error) {
		return dispatchPlugin(plugin, methodID, payload)
	})
	return nil
}

func implementedPluginMethodIDs(plugin Plugin) []uint32 {
	methods := []uint32{
		{{- range .PluginMethods }}
		{{- if not .Optional }}
		{{ .ConstName }},
		{{- end }}
		{{- end }}
	}
	{{- range .PluginMethods }}
	{{- if .Optional }}
	if _, ok := any(plugin).({{ .OptionalIfaceName }}); ok {
		methods = append(methods, {{ .ConstName }})
	}
	{{- end }}
	{{- end }}
	return methods
}

func dispatchPlugin(plugin Plugin, methodID uint32, payload []byte) ([]byte, error) {
	ctx := &PluginContext{}
	switch methodID {
	{{- range .PluginMethods }}
	case {{ .ConstName }}:
		req, err := decode{{ .RequestType }}(payload)
		if err != nil {
			return nil, err
		}
		{{- if .Optional }}
		handler, ok := any(plugin).({{ .OptionalIfaceName }})
		if !ok {
			return nil, pdkcontract.ErrMethodNotFound
		}
		resp, err := handler.{{ .GoName }}(ctx, req)
		{{- else }}
		resp, err := plugin.{{ .GoName }}(ctx, req)
		{{- end }}
		if err != nil {
			return nil, err
		}
		return encode{{ .ResponseType }}(resp)
	{{- end }}
	default:
		return nil, pdkcontract.ErrMethodNotFound
	}
}

type flatbuffersPacker interface {
	Pack(*flatbuffers.Builder) flatbuffers.UOffsetT
}

var builderPool = make(chan *flatbuffers.Builder, 32)

func acquireBuilder() *flatbuffers.Builder {
	select {
	case builder := <-builderPool:
		builder.Reset()
		return builder
	default:
		return flatbuffers.NewBuilder(128)
	}
}

func releaseBuilder(builder *flatbuffers.Builder) {
	if builder == nil {
		return
	}
	builder.Reset()
	select {
	case builderPool <- builder:
	default:
	}
}

func decodeFlatbuffer(typeName string, payload []byte, decode func([]byte) any) (_ any, err error) {
	if len(payload) < flatbuffers.SizeUint32 {
		return nil, fmt.Errorf("decode %s: invalid flatbuffer payload", typeName)
	}
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("decode %s: invalid flatbuffer payload", typeName)
		}
	}()
	return decode(payload), nil
}

{{ range .UniqueTypes }}
func encode{{ .TypeName }}(msg *{{ .TypeName }}T) ([]byte, error) {
	if msg == nil {
		msg = &{{ .TypeName }}T{}
	}
	builder := acquireBuilder()
	defer releaseBuilder(builder)
	offset := msg.Pack(builder)
	Finish{{ .TypeName }}Buffer(builder, offset)
	return append([]byte(nil), builder.FinishedBytes()...), nil
}

func withEncoded{{ .TypeName }}(msg *{{ .TypeName }}T, fn func([]byte) error) error {
	if msg == nil {
		msg = &{{ .TypeName }}T{}
	}
	builder := acquireBuilder()
	defer releaseBuilder(builder)
	offset := msg.Pack(builder)
	Finish{{ .TypeName }}Buffer(builder, offset)
	return fn(builder.FinishedBytes())
}

func decode{{ .TypeName }}(payload []byte) (*{{ .TypeName }}T, error) {
	msg, err := decode{{ .TypeName }}View(payload)
	if err != nil {
		return nil, err
	}
	return msg.UnPack(), nil
}

func decode{{ .TypeName }}View(payload []byte) (*{{ .TypeName }}, error) {
	msg, err := decodeFlatbuffer("{{ .TypeName }}", payload, func(raw []byte) any {
		return GetRootAs{{ .TypeName }}(raw, 0)
	})
	if err != nil {
		return nil, err
	}
	return msg.(*{{ .TypeName }}), nil
}

{{ end -}}
`

const flatbuffersRustLibTemplate = `// Code generated by hookr. DO NOT EDIT.

pub mod {{ .RustSchemaModule }};
pub mod hookr_plugin;

pub mod schema {
    pub use crate::{{ .RustSchemaModule }}{{ if .RustNamespacePath }}::{{ .RustNamespacePath }}{{ end }}::*;
}
`

const flatbuffersRustPluginTemplate = `// Code generated by hookr. DO NOT EDIT.

use std::boxed::Box;
use std::string::String;
use std::vec::Vec;

use crate::schema;

const ABI_MAJOR: u16 = 2;
const ABI_MINOR: u16 = 0;
pub const CONTRACT_CAPABILITIES: u64 = {{ .ContractCapabilities }};
pub const SCHEMA_HASH: [u8; 32] = [{{ join .SchemaHashLiterals ", " }}];

{{ range .PluginMethods -}}
pub const {{ .RustConstName }}: u32 = {{ .ID }};
{{ end }}
{{ if .HasHostModules }}
{{ range .HostModules }}
{{ range .Methods -}}
pub const {{ .RustConstName }}: u32 = {{ .ID }};
{{ end }}
{{ end }}
{{ end }}

pub type HookrResult<T> = Result<T, HookrError>;

#[derive(Debug, Clone)]
pub struct HookrError {
    message: String,
}

impl HookrError {
    pub fn new(message: impl Into<String>) -> Self {
        Self { message: message.into() }
    }

    pub fn method_not_found() -> Self {
        Self::new("method not found")
    }

    pub fn invalid_flatbuffer(type_name: &str) -> Self {
        Self::new(format!("decode {type_name}: invalid flatbuffer payload"))
    }

    fn as_bytes(&self) -> &[u8] {
        self.message.as_bytes()
    }
}

impl core::fmt::Display for HookrError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.write_str(&self.message)
    }
}

impl std::error::Error for HookrError {}

{{ if .HasHostModules -}}
#[derive(Default)]
pub struct PluginContext {
    {{- range .HostModules }}
    pub {{ .RustFieldName }}: {{ .RustTypeName }}Client,
    {{- end }}
}

{{ range .HostModules -}}
#[derive(Clone, Copy, Default)]
pub struct {{ .RustTypeName }}Client;

impl {{ .RustTypeName }}Client {
    {{- range .Methods }}
    pub fn {{ .RustName }}_raw(&self, payload: &[u8]) -> HookrResult<Vec<u8>> {
        host_call_raw({{ .RustConstName }}, payload)
    }

    {{- end }}
}

{{ end -}}
{{ else -}}
#[derive(Default)]
pub struct PluginContext;
{{ end }}

pub trait Plugin {
    {{- range .PluginMethods }}
    {{- if .Optional }}
    fn {{ .RustName }}<'a>(
        &mut self,
        ctx: &mut PluginContext,
        req: schema::{{ .RequestType }}<'_>,
        builder: &mut flatbuffers::FlatBufferBuilder<'a>,
    ) -> HookrResult<flatbuffers::WIPOffset<schema::{{ .ResponseType }}<'a>>> {
        let _ = (ctx, req, builder);
        Err(HookrError::method_not_found())
    }
    {{- else }}
    fn {{ .RustName }}<'a>(
        &mut self,
        ctx: &mut PluginContext,
        req: schema::{{ .RequestType }}<'_>,
        builder: &mut flatbuffers::FlatBufferBuilder<'a>,
    ) -> HookrResult<flatbuffers::WIPOffset<schema::{{ .ResponseType }}<'a>>>;
    {{- end }}
    {{- end }}
}

static mut PLUGIN: Option<Box<dyn Plugin>> = None;
static mut METHOD_BYTES: Vec<u8> = Vec::new();

pub fn register_plugin<P: Plugin + 'static>(plugin: P, optional_method_ids: &[u32]) {
    unsafe {
        PLUGIN = Some(Box::new(plugin));
        METHOD_BYTES = implemented_method_bytes(optional_method_ids);
    }
}

fn implemented_method_bytes(optional_method_ids: &[u32]) -> Vec<u8> {
    let mut method_ids = Vec::from([
        {{- range .PluginMethods }}
        {{- if not .Optional }}
        {{ .RustConstName }},
        {{- end }}
        {{- end }}
    ]);
    method_ids.extend_from_slice(optional_method_ids);
    method_ids.sort_unstable();
    method_ids.dedup();

    let mut out = Vec::with_capacity(method_ids.len() * 4);
    for method_id in method_ids {
        out.extend_from_slice(&method_id.to_le_bytes());
    }
    out
}

fn dispatch_plugin(plugin: &mut dyn Plugin, method_id: u32, payload: &[u8]) -> HookrResult<Vec<u8>> {
    let mut ctx = PluginContext::default();
    match method_id {
        {{- range .PluginMethods }}
        {{ .RustConstName }} => {
            let req = flatbuffers::root::<schema::{{ .RequestType }}>(payload)
                .map_err(|_| HookrError::invalid_flatbuffer("{{ .RequestType }}"))?;
            let mut builder = flatbuffers::FlatBufferBuilder::new();
            let response = plugin.{{ .RustName }}(&mut ctx, req, &mut builder)?;
            builder.finish(response, None);
            Ok(builder.finished_data().to_vec())
        }
        {{- end }}
        _ => Err(HookrError::method_not_found()),
    }
}

#[link(wasm_import_module = "hookr")]
extern "C" {
    fn __plugin_request(ptr: *mut u8) -> u32;
    fn __plugin_response(ptr: *const u8, len: u32);
    fn __plugin_error(ptr: *const u8, len: u32);
    fn __host_call(method_id: u32, payload_ptr: *const u8, payload_len: u32) -> u32;
    fn __host_response_len() -> u32;
    fn __host_response(ptr: *mut u8);
    fn __host_error_len() -> u32;
    fn __host_error(ptr: *mut u8);
    fn __log(ptr: *const u8, len: u32);
}

#[no_mangle]
pub extern "C" fn __plugin_call(method_id: u32, payload_len: u32) -> u32 {
    let mut payload = vec![0u8; payload_len as usize];
    let request_ptr = if payload.is_empty() {
        core::ptr::null_mut()
    } else {
        payload.as_mut_ptr()
    };
    if unsafe { __plugin_request(request_ptr) } == 0 {
        publish_plugin_error(&HookrError::new("failed to load request payload from host"));
        return 0;
    }

    let result = unsafe {
        match PLUGIN.as_mut() {
            Some(plugin) => dispatch_plugin(plugin.as_mut(), method_id, &payload),
            None => Err(HookrError::new("plugin implementation is not registered")),
        }
    };

    match result {
        Ok(response) => {
            let ptr = if response.is_empty() {
                core::ptr::null()
            } else {
                response.as_ptr()
            };
            unsafe { __plugin_response(ptr, response.len() as u32) };
            1
        }
        Err(err) => {
            publish_plugin_error(&err);
            0
        }
    }
}

#[no_mangle]
pub extern "C" fn __hookr_abi_version() -> u32 {
    ((ABI_MAJOR as u32) << 16) | ABI_MINOR as u32
}

#[no_mangle]
pub extern "C" fn __hookr_schema_hash() -> u64 {
    pack_ptr_len(SCHEMA_HASH.as_ptr() as u32, SCHEMA_HASH.len() as u32)
}

#[no_mangle]
pub extern "C" fn __hookr_capabilities() -> u64 {
    CONTRACT_CAPABILITIES
}

#[no_mangle]
pub extern "C" fn __hookr_methods() -> u64 {
    unsafe {
        if METHOD_BYTES.is_empty() {
            return 0;
        }
        pack_ptr_len(METHOD_BYTES.as_ptr() as u32, METHOD_BYTES.len() as u32)
    }
}

pub fn log(message: &str) {
    if !message.is_empty() {
        unsafe { __log(message.as_ptr(), message.len() as u32) };
    }
}

fn host_call_raw(method_id: u32, payload: &[u8]) -> HookrResult<Vec<u8>> {
    let payload_ptr = if payload.is_empty() {
        core::ptr::null()
    } else {
        payload.as_ptr()
    };
    let ok = unsafe { __host_call(method_id, payload_ptr, payload.len() as u32) };
    if ok == 0 {
        let len = unsafe { __host_error_len() };
        if len == 0 {
            return Err(HookrError::new("host call failed without an error message"));
        }
        let mut buf = vec![0u8; len as usize];
        unsafe { __host_error(buf.as_mut_ptr()) };
        let message = String::from_utf8_lossy(&buf).into_owned();
        return Err(HookrError::new(format!("Host error: {message}")));
    }

    let len = unsafe { __host_response_len() };
    let mut response = vec![0u8; len as usize];
    if len > 0 {
        unsafe { __host_response(response.as_mut_ptr()) };
    }
    Ok(response)
}

fn publish_plugin_error(err: &HookrError) {
    let bytes = err.as_bytes();
    unsafe { __plugin_error(bytes.as_ptr(), bytes.len() as u32) };
}

fn pack_ptr_len(ptr: u32, len: u32) -> u64 {
    ((ptr as u64) << 32) | len as u64
}
`

const flatbuffersZigPluginTemplate = `// Code generated by hookr. DO NOT EDIT.

const std = @import("std");

const abi_major: u16 = 2;
const abi_minor: u16 = 0;
pub const contract_capabilities: u64 = {{ .ContractCapabilities }};
pub const schema_hash = [_]u8{ {{ join .SchemaHashLiterals ", " }} };

{{ range .PluginMethods -}}
pub const {{ .RustConstName }}: u32 = {{ .ID }};
{{ end }}
{{ if .HasHostModules }}
{{ range .HostModules }}
{{ range .Methods -}}
pub const {{ .RustConstName }}: u32 = {{ .ID }};
{{ end }}
{{ end }}
{{ end }}

pub const HookrError = error{
    MethodNotFound,
    PluginNotRegistered,
    RequestUnavailable,
    HostCallFailed,
};

pub const DispatchFn = *const fn (method_id: u32, payload: []const u8) anyerror![]const u8;

var dispatch_fn: ?DispatchFn = null;
var method_bytes: []const u8 = "";

extern "hookr" fn __plugin_request(ptr: [*]u8) u32;
extern "hookr" fn __plugin_response(ptr: [*]const u8, len: u32) void;
extern "hookr" fn __plugin_error(ptr: [*]const u8, len: u32) void;
extern "hookr" fn __host_call(method_id: u32, payload_ptr: [*]const u8, payload_len: u32) u32;
extern "hookr" fn __host_response_len() u32;
extern "hookr" fn __host_response(ptr: [*]u8) void;
extern "hookr" fn __host_error_len() u32;
extern "hookr" fn __host_error(ptr: [*]u8) void;
extern "hookr" fn __log(ptr: [*]const u8, len: u32) void;

pub fn registerPlugin(dispatch: DispatchFn, implemented_method_bytes: []const u8) void {
    dispatch_fn = dispatch;
    method_bytes = implemented_method_bytes;
}

pub fn defaultImplementedMethodBytes() []const u8 {
    return &[_]u8{
        {{- range .PluginMethods }}
        {{- if not .Optional }}
        {{ .RustConstName }} & 0xff, ({{ .RustConstName }} >> 8) & 0xff, ({{ .RustConstName }} >> 16) & 0xff, ({{ .RustConstName }} >> 24) & 0xff,
        {{- end }}
        {{- end }}
    };
}

pub export fn __plugin_call(method_id: u32, payload_len: u32) bool {
    var allocator = std.heap.wasm_allocator;
    const payload = allocator.alloc(u8, payload_len) catch {
        publishPluginError("failed to allocate request payload");
        return false;
    };
    defer allocator.free(payload);

    if (__plugin_request(payload.ptr) == 0) {
        publishPluginError("failed to load request payload from host");
        return false;
    }

    const dispatch = dispatch_fn orelse {
        publishPluginError("plugin implementation is not registered");
        return false;
    };
    const response = dispatch(method_id, payload) catch |err| {
        publishPluginError(@errorName(err));
        return false;
    };
    __plugin_response(response.ptr, @intCast(response.len));
    return true;
}

pub export fn __hookr_abi_version() u32 {
    return (@as(u32, abi_major) << 16) | @as(u32, abi_minor);
}

pub export fn __hookr_schema_hash() u64 {
    return packPtrLen(@intFromPtr(&schema_hash), schema_hash.len);
}

pub export fn __hookr_capabilities() u64 {
    return contract_capabilities;
}

pub export fn __hookr_methods() u64 {
    if (method_bytes.len == 0) return 0;
    return packPtrLen(@intFromPtr(method_bytes.ptr), method_bytes.len);
}

pub fn hostCallRaw(allocator: std.mem.Allocator, method_id: u32, payload: []const u8) ![]u8 {
    if (__host_call(method_id, payload.ptr, @intCast(payload.len)) == 0) {
        const err_len = __host_error_len();
        if (err_len == 0) return HookrError.HostCallFailed;
        const err_buf = try allocator.alloc(u8, err_len);
        defer allocator.free(err_buf);
        __host_error(err_buf.ptr);
        return HookrError.HostCallFailed;
    }

    const response_len = __host_response_len();
    const response = try allocator.alloc(u8, response_len);
    if (response_len > 0) {
        __host_response(response.ptr);
    }
    return response;
}

pub fn log(message: []const u8) void {
    if (message.len > 0) {
        __log(message.ptr, @intCast(message.len));
    }
}

fn publishPluginError(message: []const u8) void {
    __plugin_error(message.ptr, @intCast(message.len));
}

fn packPtrLen(ptr: usize, len: usize) u64 {
    return (@as(u64, @intCast(ptr)) << 32) | @as(u64, @intCast(len));
}
`
