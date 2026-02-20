package main

import (
	"testing"
)

func TestParseCapnpManifest(t *testing.T) {
	schema := []byte(`
@0xadcafecafe123456;

interface Greeter {
  hello @0 (req :HelloRequest) -> (resp :HelloResponse);
  ping @1 () -> ();
}
`)

	manifest, err := parseCapnpManifest(schema)
	if err != nil {
		t.Fatalf("parseCapnpManifest returned error: %v", err)
	}
	if manifest.Name != "Greeter" {
		t.Fatalf("manifest name = %q, want Greeter", manifest.Name)
	}
	if len(manifest.Methods) != 2 {
		t.Fatalf("methods len = %d, want 2", len(manifest.Methods))
	}
	if manifest.Methods[0].Name != "hello" || manifest.Methods[0].ID != 0 {
		t.Fatalf("unexpected first method: %#v", manifest.Methods[0])
	}
	if manifest.Methods[0].Request != "HelloRequest" || manifest.Methods[0].Response != "HelloResponse" {
		t.Fatalf("unexpected first method types: %#v", manifest.Methods[0])
	}
	if manifest.Methods[1].Request != "Void" || manifest.Methods[1].Response != "Void" {
		t.Fatalf("unexpected second method void types: %#v", manifest.Methods[1])
	}
}

func TestParseProtoManifest(t *testing.T) {
	schema := []byte(`
syntax = "proto3";

service Greeter {
  rpc Hello(HelloRequest) returns (HelloResponse);
}
`)

	manifest, err := parseProtoManifest(schema, "")
	if err != nil {
		t.Fatalf("parseProtoManifest returned error: %v", err)
	}
	if manifest.Name != "Greeter" {
		t.Fatalf("manifest name = %q, want Greeter", manifest.Name)
	}
	if len(manifest.Methods) != 1 {
		t.Fatalf("methods len = %d, want 1", len(manifest.Methods))
	}
	wantID := deriveProtoMethodID("Greeter", "Hello")
	if manifest.Methods[0].ID != wantID {
		t.Fatalf("method id = %d, want %d", manifest.Methods[0].ID, wantID)
	}
}

func TestParseProtoManifestServiceFilter(t *testing.T) {
	schema := []byte(`
syntax = "proto3";

service One {
  rpc A(AReq) returns (AResp);
}

service Two {
  rpc B(BReq) returns (BResp);
}
`)

	manifest, err := parseProtoManifest(schema, "Two")
	if err != nil {
		t.Fatalf("parseProtoManifest returned error: %v", err)
	}
	if manifest.Name != "Two" {
		t.Fatalf("manifest name = %q, want Two", manifest.Name)
	}
	if len(manifest.Methods) != 1 || manifest.Methods[0].Name != "B" {
		t.Fatalf("unexpected methods: %#v", manifest.Methods)
	}
}

func TestInferManifestFromSchemaProto(t *testing.T) {
	schema := []byte(`
syntax = "proto3";
service Greeter {
  rpc Hello(HelloRequest) returns (HelloResponse);
}
`)
	manifest, err := inferManifestFromSchema("greeter.proto", schema, "")
	if err != nil {
		t.Fatalf("inferManifestFromSchema returned error: %v", err)
	}
	if manifest.Name != "Greeter" {
		t.Fatalf("manifest name = %q, want Greeter", manifest.Name)
	}
}
