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

	"github.com/mopeyjellyfish/hookr/internal/contract"
	"github.com/mopeyjellyfish/hookr/internal/flatc"
)

type flatbuffersTemplateData struct {
	PackageName          string
	ContractName         string
	SchemaPath           string
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
	InterfaceName string
	ClientName    string
	Methods       []flatbuffersMethodTemplate
}

type flatbuffersMethodTemplate struct {
	ID                uint32
	ServiceName       string
	Name              string
	GoName            string
	ConstName         string
	RequestType       string
	ResponseType      string
	Optional          bool
	OptionalIfaceName string
}

type flatbuffersTypeTemplate struct {
	TypeName string
}

func generateFlatBuffers(cfg Config) error {
	if !strings.EqualFold(cfg.Lang, "go") {
		return errors.New("flatbuffers generation currently supports only --lang go")
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
			ConstName:         "Method" + serviceExported + exported,
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
