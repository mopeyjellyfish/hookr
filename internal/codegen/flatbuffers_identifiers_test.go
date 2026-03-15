package codegen

import (
	"strings"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/contract"
)

func TestValidateFlatBuffersGoIdentifiers_MethodCollision(t *testing.T) {
	model := contract.Contract{
		PluginService: contract.Service{
			Name: "Plugin",
			Methods: []contract.Method{
				{
					ServiceName:       "Plugin",
					Name:              "get-info",
					RequestType:       "ReqA",
					RequestQualified:  "A.ReqA",
					ResponseType:      "RespA",
					ResponseQualified: "A.RespA",
				},
				{
					ServiceName:       "Plugin",
					Name:              "get_info",
					RequestType:       "ReqB",
					RequestQualified:  "A.ReqB",
					ResponseType:      "RespB",
					ResponseQualified: "A.RespB",
				},
			},
		},
	}

	err := validateFlatBuffersGoIdentifiers(model)
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "identifier collision") {
		t.Fatalf("expected identifier collision error, got %v", err)
	}
}

func TestValidateFlatBuffersGoIdentifiers_TypeCollision(t *testing.T) {
	model := contract.Contract{
		PluginService: contract.Service{
			Name: "Plugin",
			Methods: []contract.Method{
				{
					ServiceName:       "Plugin",
					Name:              "One",
					RequestType:       "Payload",
					RequestQualified:  "Foo.Payload",
					ResponseType:      "Result",
					ResponseQualified: "Foo.Result",
				},
				{
					ServiceName:       "Plugin",
					Name:              "Two",
					RequestType:       "Payload",
					RequestQualified:  "Bar.Payload",
					ResponseType:      "Other",
					ResponseQualified: "Bar.Other",
				},
			},
		},
	}

	err := validateFlatBuffersGoIdentifiers(model)
	if err == nil {
		t.Fatal("expected type collision error")
	}
	if !strings.Contains(err.Error(), "type helper collision") {
		t.Fatalf("expected type helper collision error, got %v", err)
	}
}

func TestValidateFlatBuffersGoIdentifiers_NoCollision(t *testing.T) {
	model := contract.Contract{
		PluginService: contract.Service{
			Name: "Plugin",
			Methods: []contract.Method{
				{
					ServiceName:       "Plugin",
					Name:              "GetInfo",
					RequestType:       "Empty",
					RequestQualified:  "Hookr.Empty",
					ResponseType:      "Info",
					ResponseQualified: "Hookr.Info",
				},
			},
		},
		HostServices: []contract.Service{
			{
				Name: "Rng",
				Methods: []contract.Method{
					{
						ServiceName:       "Rng",
						Name:              "Int",
						RequestType:       "RngIntRequest",
						RequestQualified:  "Hookr.RngIntRequest",
						ResponseType:      "RngIntResponse",
						ResponseQualified: "Hookr.RngIntResponse",
					},
				},
			},
		},
	}

	if err := validateFlatBuffersGoIdentifiers(model); err != nil {
		t.Fatalf("unexpected collision error: %v", err)
	}
}
