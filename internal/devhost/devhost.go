package devhost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/mopeyjellyfish/hookr/internal/contract"
	"github.com/mopeyjellyfish/hookr/internal/flatc"
	hookrruntime "github.com/mopeyjellyfish/hookr/runtime"
)

type Fixture struct {
	Methods map[string]MethodFixture
}

type MethodFixture struct {
	Response  json.RawMessage
	Responses []json.RawMessage
}

func LoadFixture(path string) (Fixture, error) {
	if path == "" {
		return Fixture{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, fmt.Errorf("read host fixture: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Fixture{}, fmt.Errorf("parse host fixture: %w", err)
	}
	out := Fixture{Methods: make(map[string]MethodFixture, len(raw))}
	for methodName, value := range raw {
		fixture, err := parseMethodFixture(value)
		if err != nil {
			return Fixture{}, fmt.Errorf("parse fixture for %s: %w", methodName, err)
		}
		out.Methods[methodName] = fixture
	}
	return out, nil
}

func BindHostMethods(
	model contract.Contract,
	runner *flatc.Runner,
	schemaPath string,
	includePaths []string,
	fixture Fixture,
) []hookrruntime.HostMethod {
	if model.HostService == nil {
		return nil
	}
	methods := make([]hookrruntime.HostMethod, 0, len(model.HostService.Methods))
	for _, method := range model.HostService.Methods {
		responder := newResponder(
			method,
			runner,
			schemaPath,
			includePaths,
			fixture.Methods[method.Name],
		)
		methods = append(methods, hookrruntime.HostFnMethod(method.ID, responder.Call))
	}
	return methods
}

type responder struct {
	method       contract.Method
	runner       *flatc.Runner
	schemaPath   string
	includePaths []string
	responses    []json.RawMessage
	mu           sync.Mutex
}

func newResponder(
	method contract.Method,
	runner *flatc.Runner,
	schemaPath string,
	includePaths []string,
	fixture MethodFixture,
) *responder {
	responses := fixture.Responses
	if len(responses) == 0 && len(fixture.Response) > 0 {
		responses = []json.RawMessage{fixture.Response}
	}
	return &responder{
		method:       method,
		runner:       runner,
		schemaPath:   schemaPath,
		includePaths: includePaths,
		responses:    responses,
	}
}

func (r *responder) Call(_ context.Context, _ []byte) ([]byte, error) {
	if r == nil {
		return nil, errors.New("host responder is not configured")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.responses) == 0 {
		return nil, fmt.Errorf("no fixture response configured for host method %s", r.method.Name)
	}
	response := r.responses[0]
	if len(r.responses) > 1 {
		r.responses = r.responses[1:]
	}
	data, err := r.runner.EncodeJSON(
		r.schemaPath,
		r.includePaths,
		r.method.ResponseQualified,
		response,
	)
	if err != nil {
		return nil, fmt.Errorf("encode fixture response for host method %s: %w", r.method.Name, err)
	}
	return data, nil
}

func parseMethodFixture(raw json.RawMessage) (MethodFixture, error) {
	var obj struct {
		Response  json.RawMessage   `json:"response"`
		Responses []json.RawMessage `json:"responses"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil &&
		(len(obj.Response) > 0 || len(obj.Responses) > 0) {
		return MethodFixture{Response: obj.Response, Responses: obj.Responses}, nil
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		return MethodFixture{Responses: arr}, nil
	}

	return MethodFixture{Response: raw}, nil
}
