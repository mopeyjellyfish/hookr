package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

type methodManifest struct {
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	Request  string `json:"request"`
	Response string `json:"response"`
}

type contractManifest struct {
	Name    string           `json:"name"`
	Methods []methodManifest `json:"methods"`
}

type methodTemplate struct {
	ID              uint32
	Name            string
	RequestType     string
	ResponseType    string
	ConstName       string
	CapabilityConst string
	CapabilityExpr  string
	RuntimeConst    string
	PDKConst        string
}

type templateData struct {
	PackageName              string
	ContractName             string
	Codec                    string
	SchemaPath               string
	SchemaHashHex            string
	SchemaHash               []byte
	AllMethodsCapabilityExpr string
	Methods                  []methodTemplate
}

func main() {
	var (
		schemaPath   string
		manifestPath string
		protoService string
		outDir       string
		packageName  string
		contractName string
		codec        string
	)

	flag.StringVar(&schemaPath, "schema", "", "path to schema file (required)")
	flag.StringVar(&manifestPath, "manifest", "", "path to JSON contract manifest (optional)")
	flag.StringVar(&protoService, "service", "", "proto service name to generate (optional; defaults to first service)")
	flag.StringVar(&outDir, "out", "", "output directory (required)")
	flag.StringVar(&packageName, "package", "", "generated Go package name (required)")
	flag.StringVar(&contractName, "name", "", "contract name override (optional)")
	flag.StringVar(&codec, "codec", "capnp", "wire codec label (e.g. capnp, protobuf)")
	flag.Parse()

	if schemaPath == "" || outDir == "" || packageName == "" {
		fmt.Fprintln(os.Stderr, "usage: hookr-gen -schema <file> -out <dir> -package <name> [-manifest contract.json] [-name contract] [-codec capnp]")
		os.Exit(2)
	}

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read schema: %v\n", err)
		os.Exit(1)
	}
	hash := sha256.Sum256(schemaBytes)
	hashHex := hex.EncodeToString(hash[:])

	manifest := contractManifest{}
	if manifestPath != "" {
		manifestBytes, err := os.ReadFile(manifestPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read manifest: %v\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			fmt.Fprintf(os.Stderr, "parse manifest: %v\n", err)
			os.Exit(1)
		}
	}

	if len(manifest.Methods) == 0 || manifest.Name == "" {
		inferred, err := inferManifestFromSchema(schemaPath, schemaBytes, protoService)
		if err != nil {
			fmt.Fprintf(os.Stderr, "infer methods from schema: %v\n", err)
			os.Exit(1)
		}
		if manifest.Name == "" {
			manifest.Name = inferred.Name
		}
		if len(manifest.Methods) == 0 {
			manifest.Methods = inferred.Methods
		}
	}

	if len(manifest.Methods) == 0 {
		fmt.Fprintln(os.Stderr, "no methods found; provide -manifest or use a schema with methods")
		os.Exit(1)
	}

	name := contractName
	if name == "" {
		name = manifest.Name
	}
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(schemaPath), filepath.Ext(schemaPath))
	}

	methods := make([]methodTemplate, 0, len(manifest.Methods))
	allCapabilities := make([]string, 0, len(manifest.Methods))
	seenIDs := make(map[uint32]string, len(manifest.Methods))
	for i, m := range manifest.Methods {
		if m.Name == "" {
			fmt.Fprintln(os.Stderr, "manifest method name cannot be empty")
			os.Exit(1)
		}
		if prior, ok := seenIDs[m.ID]; ok {
			fmt.Fprintf(os.Stderr, "duplicate method id %d (%s, %s)\n", m.ID, prior, m.Name)
			os.Exit(1)
		}
		seenIDs[m.ID] = m.Name
		constName := "Method" + toExportedIdentifier(m.Name)
		capabilityConst := "Capability" + toExportedIdentifier(m.Name)
		capabilityExpr := "0"
		if i < 63 {
			capabilityExpr = fmt.Sprintf("uint64(1) << %d", i)
		}
		if capabilityExpr != "0" {
			allCapabilities = append(allCapabilities, capabilityConst)
		}
		methods = append(methods, methodTemplate{
			ID:              m.ID,
			Name:            m.Name,
			RequestType:     m.Request,
			ResponseType:    m.Response,
			ConstName:       constName,
			CapabilityConst: capabilityConst,
			CapabilityExpr:  capabilityExpr,
			RuntimeConst:    "runtimecontract.MethodID(" + constName + ")",
			PDKConst:        "pdkcontract.MethodID(" + constName + ")",
		})
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].ID < methods[j].ID })

	allMethodsCapabilityExpr := "0"
	if len(allCapabilities) > 0 {
		allMethodsCapabilityExpr = strings.Join(allCapabilities, " | ")
	}

	td := templateData{
		PackageName:              packageName,
		ContractName:             name,
		Codec:                    codec,
		SchemaPath:               schemaPath,
		SchemaHashHex:            hashHex,
		SchemaHash:               hash[:],
		AllMethodsCapabilityExpr: allMethodsCapabilityExpr,
		Methods:                  methods,
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}

	writes := []struct {
		name string
		tpl  string
	}{
		{"contract_meta_gen.go", metaTemplate},
		{"runtime_glue_gen.go", runtimeTemplate},
		{"pdk_glue_gen.go", pdkTemplate},
	}

	for _, write := range writes {
		path := filepath.Join(outDir, write.name)
		if err := renderTemplate(path, write.tpl, td); err != nil {
			fmt.Fprintf(os.Stderr, "generate %s: %v\n", write.name, err)
			os.Exit(1)
		}
	}
}

func renderTemplate(path string, tpl string, data templateData) error {
	t, err := template.New(filepath.Base(path)).Parse(tpl)
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

func toExportedIdentifier(s string) string {
	if s == "" {
		return "Unnamed"
	}
	var b strings.Builder
	upperNext := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if upperNext {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(r)
			}
			upperNext = false
			continue
		}
		upperNext = true
	}
	out := b.String()
	if out == "" {
		return "Unnamed"
	}
	if unicode.IsDigit(rune(out[0])) {
		return "M" + out
	}
	return out
}

func inferManifestFromSchema(schemaPath string, schemaBytes []byte, protoService string) (contractManifest, error) {
	ext := strings.ToLower(filepath.Ext(schemaPath))
	switch ext {
	case ".capnp":
		return parseCapnpManifest(schemaBytes)
	case ".proto":
		return parseProtoManifest(schemaBytes, protoService)
	default:
		return contractManifest{}, fmt.Errorf("unsupported schema extension %q", ext)
	}
}

func parseCapnpManifest(schemaBytes []byte) (contractManifest, error) {
	var (
		manifest         contractManifest
		inInterface      bool
		interfaceDepth   int
		interfaceNameSet bool
	)
	reInterface := regexp.MustCompile(`^\s*interface\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`)
	reMethod := regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*@([0-9]+)\s*\(([^)]*)\)\s*->\s*\(([^)]*)\)\s*;`)

	sc := bufio.NewScanner(strings.NewReader(string(schemaBytes)))
	for sc.Scan() {
		line := sc.Text()
		if !inInterface {
			if match := reInterface.FindStringSubmatch(line); len(match) == 2 {
				manifest.Name = match[1]
				interfaceNameSet = true
				inInterface = true
				interfaceDepth = 1
				continue
			}
			continue
		}

		interfaceDepth += strings.Count(line, "{")
		interfaceDepth -= strings.Count(line, "}")
		if interfaceDepth <= 0 {
			break
		}

		match := reMethod.FindStringSubmatch(line)
		if len(match) != 5 {
			continue
		}

		id64, err := strconv.ParseUint(match[2], 10, 32)
		if err != nil {
			return contractManifest{}, fmt.Errorf("invalid capnp method id %q: %w", match[2], err)
		}

		manifest.Methods = append(manifest.Methods, methodManifest{
			ID:       uint32(id64),
			Name:     match[1],
			Request:  firstFieldType(match[3]),
			Response: firstFieldType(match[4]),
		})
	}
	if err := sc.Err(); err != nil {
		return contractManifest{}, err
	}
	if !interfaceNameSet {
		return contractManifest{}, errors.New("no capnp interface found")
	}
	if len(manifest.Methods) == 0 {
		return contractManifest{}, errors.New("no methods found in capnp interface")
	}
	return manifest, nil
}

func parseProtoManifest(schemaBytes []byte, serviceFilter string) (contractManifest, error) {
	var (
		manifest      contractManifest
		inService     bool
		serviceDepth  int
		currentSvc    string
		selectedFound bool
	)
	reService := regexp.MustCompile(`^\s*service\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`)
	reRPC := regexp.MustCompile(`^\s*rpc\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*([A-Za-z0-9_.]+)\s*\)\s*returns\s*\(\s*([A-Za-z0-9_.]+)\s*\)\s*;`)

	sc := bufio.NewScanner(strings.NewReader(string(schemaBytes)))
	for sc.Scan() {
		line := sc.Text()
		if !inService {
			match := reService.FindStringSubmatch(line)
			if len(match) != 2 {
				continue
			}
			currentSvc = match[1]
			if serviceFilter != "" && currentSvc != serviceFilter {
				inService = true
				serviceDepth = 1
				continue
			}
			manifest.Name = currentSvc
			inService = true
			serviceDepth = 1
			selectedFound = true
			continue
		}

		serviceDepth += strings.Count(line, "{")
		serviceDepth -= strings.Count(line, "}")

		if manifest.Name == currentSvc {
			match := reRPC.FindStringSubmatch(line)
			if len(match) == 4 {
				id := deriveProtoMethodID(currentSvc, match[1])
				manifest.Methods = append(manifest.Methods, methodManifest{
					ID:       id,
					Name:     match[1],
					Request:  match[2],
					Response: match[3],
				})
			}
		}

		if serviceDepth <= 0 {
			if manifest.Name == currentSvc && len(manifest.Methods) > 0 {
				break
			}
			inService = false
			currentSvc = ""
		}
	}
	if err := sc.Err(); err != nil {
		return contractManifest{}, err
	}
	if serviceFilter != "" && !selectedFound {
		return contractManifest{}, fmt.Errorf("proto service %q not found", serviceFilter)
	}
	if manifest.Name == "" {
		return contractManifest{}, errors.New("no proto service found")
	}
	if len(manifest.Methods) == 0 {
		return contractManifest{}, fmt.Errorf("no rpc methods found in proto service %q", manifest.Name)
	}
	return manifest, nil
}

func firstFieldType(fields string) string {
	trimmed := strings.TrimSpace(fields)
	if trimmed == "" {
		return "Void"
	}
	parts := strings.Split(trimmed, ",")
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		idx := strings.Index(p, ":")
		if idx == -1 {
			continue
		}
		typeName := strings.TrimSpace(p[idx+1:])
		if typeName != "" {
			return typeName
		}
	}
	return "Void"
}

func deriveProtoMethodID(service, method string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(service))
	_, _ = h.Write([]byte{'.'})
	_, _ = h.Write([]byte(method))
	id := h.Sum32()
	if id == 0 {
		return 1
	}
	return id
}

const metaTemplate = `// Code generated by hookr-gen. DO NOT EDIT.
// Source schema: {{ .SchemaPath }}

package {{ .PackageName }}

type Method struct {
	ID           uint32
	Name         string
	RequestType  string
	ResponseType string
}

const (
	ContractName  = {{ printf "%q" .ContractName }}
	ContractCodec = {{ printf "%q" .Codec }}
	SchemaHashHex = {{ printf "%q" .SchemaHashHex }}
{{- range .Methods }}
	{{ .ConstName }} uint32 = {{ .ID }}
{{- end }}

{{- range .Methods }}
	{{ .CapabilityConst }} uint64 = {{ .CapabilityExpr }}
{{- end }}
	ContractCapabilities uint64 = {{ .AllMethodsCapabilityExpr }}
)

var Methods = []Method{
{{- range .Methods }}
	{ID: {{ .ConstName }}, Name: {{ printf "%q" .Name }}, RequestType: {{ printf "%q" .RequestType }}, ResponseType: {{ printf "%q" .ResponseType }}},
{{- end }}
}
`

const runtimeTemplate = `// Code generated by hookr-gen. DO NOT EDIT.
//go:build !wasip1

package {{ .PackageName }}

import (
	"context"
	"errors"
	"fmt"

	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/mopeyjellyfish/hookr/runtime/module"
)

var runtimeSchemaHash = [runtimecontract.SchemaHashLen]byte{
{{- range $i, $b := .SchemaHash }}
	{{ $b }},
{{- end }}
}

func RuntimeHandshake() runtimecontract.Handshake {
	hs := runtimecontract.NewHandshake(runtimeSchemaHash)
	hs.Capabilities = ContractCapabilities
	return hs
}

func RuntimeSchema() runtimecontract.Schema {
	return runtimecontract.Schema{
		Name:       ContractName,
		SchemaHash: runtimeSchemaHash,
		Capabilities: ContractCapabilities,
		Methods: []runtimecontract.Method{
{{- range .Methods }}
			{
				ID:           {{ .RuntimeConst }},
				Name:         {{ printf "%q" .Name }},
				RequestType:  {{ printf "%q" .RequestType }},
				ResponseType: {{ printf "%q" .ResponseType }},
			},
{{- end }}
		},
	}
}

type RuntimeMethodHandlers struct {
{{- range .Methods }}
	{{ .ConstName }} func(ctx context.Context, payload []byte) ([]byte, error)
{{- end }}
}

func RuntimeCallHandlerV2(h RuntimeMethodHandlers) module.CallHandlerV2 {
	return func(ctx context.Context, methodID uint32, payload []byte) ([]byte, error) {
		switch methodID {
{{- range .Methods }}
		case {{ .ConstName }}:
			if h.{{ .ConstName }} == nil {
				return nil, errors.New("runtime handler {{ .ConstName }} is nil")
			}
			return h.{{ .ConstName }}(ctx, payload)
{{- end }}
		default:
			return nil, fmt.Errorf("unknown method id %d", methodID)
		}
	}
}

{{- range .Methods }}
func BindRuntime{{ .ConstName }}[Req any, Resp any](
	decodeReq runtimecontract.Decoder[Req],
	encodeResp runtimecontract.Encoder[Resp],
	fn func(ctx context.Context, req Req) (Resp, error),
) func(context.Context, []byte) ([]byte, error) {
	return runtimecontract.BindHostMethod(decodeReq, encodeResp, fn)
}

func Host{{ .ConstName }}[Req any, Resp any](
	decodeReq runtimecontract.Decoder[Req],
	encodeResp runtimecontract.Encoder[Resp],
	fn func(ctx context.Context, req Req) (Resp, error),
) hookrruntime.HostMethod {
	return hookrruntime.HostFnMethod({{ .ConstName }}, runtimecontract.BindHostMethod(decodeReq, encodeResp, fn))
}

func CallPlugin{{ .ConstName }}[Req any, Resp any](
	ctx context.Context,
	rt *hookrruntime.Runtime,
	req Req,
	encodeReq runtimecontract.Encoder[Req],
	decodeResp runtimecontract.Decoder[Resp],
) (Resp, error) {
	var zero Resp
	payload, err := encodeReq(req)
	if err != nil {
		return zero, err
	}
	fn, err := hookrruntime.PluginFnMethod(rt, {{ .ConstName }})
	if err != nil {
		return zero, err
	}
	data, err := fn.Call(ctx, payload)
	if err != nil {
		return zero, err
	}
	return decodeResp(data)
}
{{- end }}
`

const pdkTemplate = `// Code generated by hookr-gen. DO NOT EDIT.
//go:build wasip1

package {{ .PackageName }}

import (
	"errors"
	"fmt"

	"github.com/mopeyjellyfish/hookr/pdk"
	pdkcontract "github.com/mopeyjellyfish/hookr/pdk/contract"
)

var pdkSchemaHash = [pdkcontract.SchemaHashLen]byte{
{{- range $i, $b := .SchemaHash }}
	{{ $b }},
{{- end }}
}

func PDKHandshake() pdkcontract.Handshake {
	hs := pdkcontract.NewHandshake(pdkSchemaHash)
	hs.Capabilities = ContractCapabilities
	return hs
}

func EnablePDKHandshake() {
	pdkcontract.EnableHandshakeWithCapabilities(pdkSchemaHash, ContractCapabilities)
}

func PDKSchema() pdkcontract.Schema {
	return pdkcontract.Schema{
		Name:       ContractName,
		SchemaHash: pdkSchemaHash,
		Capabilities: ContractCapabilities,
		Methods: []pdkcontract.Method{
{{- range .Methods }}
			{
				ID:           {{ .PDKConst }},
				Name:         {{ printf "%q" .Name }},
				RequestType:  {{ printf "%q" .RequestType }},
				ResponseType: {{ printf "%q" .ResponseType }},
			},
{{- end }}
		},
	}
}

{{- range .Methods }}
func BindPlugin{{ .ConstName }}[Req any, Resp any](
	decodeReq pdkcontract.Decoder[Req],
	encodeResp pdkcontract.Encoder[Resp],
	fn func(req Req) (Resp, error),
) func([]byte) ([]byte, error) {
	return pdkcontract.BindPluginMethod(decodeReq, encodeResp, fn)
}

func RegisterPlugin{{ .ConstName }}[Req any, Resp any](
	decodeReq pdkcontract.Decoder[Req],
	encodeResp pdkcontract.Encoder[Resp],
	fn func(req Req) (Resp, error),
) {
	pdk.FnMethod({{ .ConstName }}, pdkcontract.BindPluginMethod(decodeReq, encodeResp, fn))
}

func CallHost{{ .ConstName }}[Req any, Resp any](
	req Req,
	encodeReq pdkcontract.Encoder[Req],
	decodeResp pdkcontract.Decoder[Resp],
) (Resp, error) {
	var zero Resp
	payload, err := encodeReq(req)
	if err != nil {
		return zero, err
	}
	data, err := pdk.HostCallMethod({{ .ConstName }}, payload)
	if err != nil {
		return zero, err
	}
	return decodeResp(data)
}
{{- end }}

type PluginMethodHandlers struct {
{{- range .Methods }}
	{{ .ConstName }} func(payload []byte) ([]byte, error)
{{- end }}
}

func SetPluginMethodDispatcher(h PluginMethodHandlers) {
	pdk.SetMethodDispatcher(func(methodID uint32, payload []byte) ([]byte, error) {
		switch methodID {
{{- range .Methods }}
		case {{ .ConstName }}:
			if h.{{ .ConstName }} == nil {
				return nil, errors.New("plugin handler {{ .ConstName }} is nil")
			}
			return h.{{ .ConstName }}(payload)
{{- end }}
		default:
			return nil, fmt.Errorf("unknown method id %d", methodID)
		}
	})
}
`
