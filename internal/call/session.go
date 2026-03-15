package call

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mopeyjellyfish/hookr/internal/contract"
	"github.com/mopeyjellyfish/hookr/internal/devhost"
	"github.com/mopeyjellyfish/hookr/internal/flatbuffers/reflection"
	"github.com/mopeyjellyfish/hookr/internal/flatc"
	"github.com/mopeyjellyfish/hookr/internal/schemautil"
	"github.com/mopeyjellyfish/hookr/internal/trustopts"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
)

type Session struct {
	cfg         Config
	runner      *flatc.Runner
	contract    contract.Contract
	schema      *reflection.Schema
	rt          *hookrruntime.Runtime
	hostFixture devhost.Fixture
}

type PreparedCall struct {
	Method  contract.Method
	Payload []byte
}

type Result struct {
	Method        contract.Method
	RequestBytes  int
	ResponseBytes int
	Duration      time.Duration
	ResponseJSON  []byte
}

type BinaryResult struct {
	Method        contract.Method
	RequestBytes  int
	ResponseBytes int
	Duration      time.Duration
	Response      []byte
}

type DebugInfo struct {
	ContractName        string
	SchemaPath          string
	PluginPath          string
	HostFixturePath     string
	SchemaHash          string
	PluginSchemaHash    string
	SchemaHashMatch     bool
	ABIVersion          string
	Capabilities        uint64
	PluginMethodIDs     []uint32
	ContractMethodCount int
}

func NewSession(cfg Config) (*Session, error) {
	if cfg.SchemaPath == "" {
		return nil, errors.New("schema path is required")
	}
	if cfg.PluginPath == "" {
		return nil, errors.New("plugin path is required")
	}
	runner, model, schema, err := schemautil.LoadWithReflection(schemautil.Config{
		SchemaPath:        cfg.SchemaPath,
		FlatcPath:         cfg.FlatcPath,
		IncludePaths:      cfg.IncludePaths,
		Package:           cfg.Package,
		PluginService:     cfg.PluginService,
		HostService:       cfg.HostService,
		OptionalAttribute: cfg.OptionalAttribute,
	})
	if err != nil {
		return nil, err
	}
	hostFixture, err := devhost.LoadFixture(cfg.HostFixturePath)
	if err != nil {
		return nil, err
	}
	fileOptions, err := trustopts.Build(cfg.Hash, cfg.AllowUnsigned)
	if err != nil {
		return nil, err
	}
	opts := []hookrruntime.Option{
		hookrruntime.WithFile(cfg.PluginPath, fileOptions...),
		hookrruntime.WithContractSchema(runtimeSchema(model)),
	}
	hostMethods := devhost.BindHostMethods(
		model,
		runner,
		cfg.SchemaPath,
		cfg.IncludePaths,
		hostFixture,
	)
	if len(hostMethods) > 0 {
		opts = append(opts, hookrruntime.WithHostMethodFns(hostMethods...))
	}
	rt, err := hookrruntime.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return &Session{
		cfg:         cfg,
		runner:      runner,
		contract:    model,
		schema:      schema,
		rt:          rt,
		hostFixture: hostFixture,
	}, nil
}

func (s *Session) Close() error {
	if s == nil || s.rt == nil {
		return nil
	}
	return s.rt.Close(context.Background())
}

func (s *Session) Contract() contract.Contract {
	if s == nil {
		return contract.Contract{}
	}
	return s.contract
}

func (s *Session) Method(name string) (contract.Method, bool) {
	if s == nil {
		return contract.Method{}, false
	}
	return s.contract.PluginMethod(name)
}

func (s *Session) DefaultRequestJSON(methodName string) (string, error) {
	method, ok := s.Method(methodName)
	if !ok {
		return "", fmt.Errorf("plugin method %q not found in contract", methodName)
	}
	if template, err := buildTemplateJSON(s.schema, method.RequestQualified); err == nil {
		return prettyJSON(template), nil
	}
	encoded, err := s.runner.EncodeJSON(
		s.cfg.SchemaPath,
		s.cfg.IncludePaths,
		method.RequestQualified,
		[]byte("{}"),
	)
	if err != nil {
		return "", fmt.Errorf("encode default request for %s: %w", method.Name, err)
	}
	decoded, err := s.runner.DecodeJSON(
		s.cfg.SchemaPath,
		s.cfg.IncludePaths,
		method.RequestQualified,
		encoded,
	)
	if err != nil {
		return "", fmt.Errorf("decode default request for %s: %w", method.Name, err)
	}
	return prettyJSON(decoded), nil
}

func (s *Session) InvokeJSON(
	ctx context.Context,
	methodName string,
	rawJSON []byte,
) (Result, error) {
	prepared, err := s.PrepareJSON(methodName, rawJSON)
	if err != nil {
		return Result{}, err
	}
	rawResult, err := s.InvokePrepared(ctx, prepared)
	if err != nil {
		return Result{}, err
	}
	respJSON, err := s.runner.DecodeJSON(
		s.cfg.SchemaPath,
		s.cfg.IncludePaths,
		rawResult.Method.ResponseQualified,
		rawResult.Response,
	)
	if err != nil {
		return Result{}, fmt.Errorf("decode response for %s: %w", rawResult.Method.Name, err)
	}
	return Result{
		Method:        rawResult.Method,
		RequestBytes:  rawResult.RequestBytes,
		ResponseBytes: rawResult.ResponseBytes,
		Duration:      rawResult.Duration,
		ResponseJSON:  ensureTrailingNewline(respJSON),
	}, nil
}

func (s *Session) PrepareJSON(methodName string, rawJSON []byte) (PreparedCall, error) {
	method, ok := s.Method(methodName)
	if !ok {
		return PreparedCall{}, fmt.Errorf("plugin method %q not found in contract", methodName)
	}
	payload, err := s.runner.EncodeJSON(
		s.cfg.SchemaPath,
		s.cfg.IncludePaths,
		method.RequestQualified,
		rawJSON,
	)
	if err != nil {
		return PreparedCall{}, fmt.Errorf("encode request for %s: %w", method.Name, err)
	}
	return PreparedCall{
		Method:  method,
		Payload: payload,
	}, nil
}

func (s *Session) InvokePrepared(ctx context.Context, prepared PreparedCall) (BinaryResult, error) {
	method := prepared.Method
	if !s.rt.HasPluginMethodID(method.ID) {
		return BinaryResult{}, fmt.Errorf(
			"plugin does not implement method %s (%d)",
			method.Name,
			method.ID,
		)
	}
	start := time.Now()
	response, err := s.rt.InvokeMethod(ctx, method.ID, prepared.Payload)
	if err != nil {
		return BinaryResult{}, err
	}
	return BinaryResult{
		Method:        method,
		RequestBytes:  len(prepared.Payload),
		ResponseBytes: len(response),
		Duration:      time.Since(start),
		Response:      response,
	}, nil
}

func (s *Session) DebugInfo() DebugInfo {
	if s == nil {
		return DebugInfo{}
	}
	info := DebugInfo{
		ContractName:        s.contract.Name,
		SchemaPath:          s.contract.SchemaPath,
		PluginPath:          s.cfg.PluginPath,
		HostFixturePath:     s.cfg.HostFixturePath,
		SchemaHash:          hex.EncodeToString(s.contract.SchemaHash[:]),
		ContractMethodCount: len(s.contract.PluginService.Methods),
	}
	if s.rt == nil {
		return info
	}
	if hs, ok := s.rt.PluginHandshake(); ok {
		info.PluginSchemaHash = hex.EncodeToString(hs.SchemaHash[:])
		info.SchemaHashMatch = hs.SchemaHash == s.contract.SchemaHash
		info.ABIVersion = fmt.Sprintf("%d.%d", hs.ABIMajor, hs.ABIMinor)
		info.Capabilities = hs.Capabilities
	}
	info.PluginMethodIDs = s.rt.PluginMethodIDs()
	return info
}

func prettyJSON(raw []byte) string {
	trimmed := ensureTrailingNewline(raw)
	var out bytes.Buffer
	if err := json.Indent(&out, trimmed, "", "  "); err == nil {
		return out.String()
	}
	return string(trimmed)
}

func ensureTrailingNewline(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}\n")
	}
	if raw[len(raw)-1] == '\n' {
		return raw
	}
	return append(append([]byte(nil), raw...), '\n')
}

func runtimeSchema(model contract.Contract) runtimecontract.Schema {
	methods := make([]runtimecontract.Method, 0, len(model.PluginService.Methods))
	for _, method := range model.PluginService.Methods {
		methods = append(methods, runtimecontract.Method{
			ID:           runtimecontract.MethodID(method.ID),
			Name:         method.Name,
			RequestType:  method.RequestType,
			ResponseType: method.ResponseType,
			Optional:     method.Optional,
		})
	}
	return runtimecontract.Schema{
		Name:         model.Name,
		SchemaHash:   model.SchemaHash,
		Capabilities: 0,
		Methods:      methods,
	}
}
