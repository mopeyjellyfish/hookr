package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mopeyjellyfish/hookr/internal/flatc"
)

func TestLoadContractHashIncludesReachableTypeShape(t *testing.T) {
	t.Parallel()

	runner, err := flatc.New("")
	if err != nil {
		t.Skipf("flatc not available: %v", err)
	}

	tmpDir := t.TempDir()
	schemaA := filepath.Join(tmpDir, "a.fbs")
	schemaB := filepath.Join(tmpDir, "b.fbs")
	writeFile(t, schemaA, `
namespace Hookr.Test.Hash;

table Request {
  value:string;
}

table Response {
  ok:bool;
}

rpc_service Plugin {
  Echo(Request):Response;
}
`)
	writeFile(t, schemaB, `
namespace Hookr.Test.Hash;

table Request {
  value:string;
  count:uint32;
}

table Response {
  ok:bool;
}

rpc_service Plugin {
  Echo(Request):Response;
}
`)

	contractA := loadContractForTest(t, runner, schemaA, "hashcontract")
	contractB := loadContractForTest(t, runner, schemaB, "hashcontract")
	if contractA.SchemaHash == contractB.SchemaHash {
		t.Fatalf("schema hash should change when reachable type layout changes")
	}
}

func TestLoadContractHashIgnoresDocumentationOnlyChanges(t *testing.T) {
	t.Parallel()

	runner, err := flatc.New("")
	if err != nil {
		t.Skipf("flatc not available: %v", err)
	}

	tmpDir := t.TempDir()
	schemaA := filepath.Join(tmpDir, "a.fbs")
	schemaB := filepath.Join(tmpDir, "b.fbs")
	writeFile(t, schemaA, `
namespace Hookr.Test.Hash;

table Request {
  value:string;
}

table Response {
  ok:bool;
}

rpc_service Plugin {
  Echo(Request):Response;
}
`)
	writeFile(t, schemaB, `
namespace Hookr.Test.Hash;

/// Documentation should not affect the contract hash.
table Request {
  /// Same field, different docs.
  value:string;
}

table Response {
  ok:bool;
}

rpc_service Plugin {
  /// Same method, different docs.
  Echo(Request):Response;
}
`)

	contractA := loadContractForTest(t, runner, schemaA, "hashcontract")
	contractB := loadContractForTest(t, runner, schemaB, "hashcontract")
	if contractA.SchemaHash != contractB.SchemaHash {
		t.Fatalf("schema hash should remain stable for documentation-only changes")
	}
}

func TestLoadContractDiscoversHostServices(t *testing.T) {
	t.Parallel()

	runner, err := flatc.New("")
	if err != nil {
		t.Skipf("flatc not available: %v", err)
	}

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "modules.fbs")
	writeFile(t, schemaPath, `
namespace Hookr.Test.Modules;

table UpdateRequest {}
table UpdateResponse { ok:bool; }
table IntRequest { min:int32; max:int32; }
table IntResponse { value:int32; }
table FloatRequest {}
table FloatResponse { value:float32; }
table PresenceRequest { key:string; }
table PresenceResponse { online:bool; }

rpc_service Presence {
  Get(PresenceRequest):PresenceResponse;
}

rpc_service Plugin {
  Update(UpdateRequest):UpdateResponse;
}

rpc_service Rng {
  Float(FloatRequest):FloatResponse;
  Int(IntRequest):IntResponse;
}
`)

	model := loadContractForTest(t, runner, schemaPath, "modules")
	if got := len(model.HostServices); got != 2 {
		t.Fatalf("host services = %d, want 2", got)
	}
	if model.HostServices[0].Name != "Presence" || model.HostServices[1].Name != "Rng" {
		t.Fatalf("host services = %#v, want Presence,Rng", model.HostServices)
	}
	if _, ok := model.HostMethod("Rng", "Int"); !ok {
		t.Fatal("expected Rng.Int host method")
	}
	if _, ok := model.HostMethod("Presence", "Get"); !ok {
		t.Fatal("expected Presence.Get host method")
	}
	if _, ok := model.HostMethod("Missing", "Get"); ok {
		t.Fatal("unexpected host method for missing service")
	}
}

func TestLoadContractHashStableForHostServiceOrder(t *testing.T) {
	t.Parallel()

	runner, err := flatc.New("")
	if err != nil {
		t.Skipf("flatc not available: %v", err)
	}

	tmpDir := t.TempDir()
	schemaA := filepath.Join(tmpDir, "a.fbs")
	schemaB := filepath.Join(tmpDir, "b.fbs")
	const common = `
namespace Hookr.Test.Modules;

table UpdateRequest {}
table UpdateResponse { ok:bool; }
table IntRequest { min:int32; max:int32; }
table IntResponse { value:int32; }
table PresenceRequest { key:string; }
table PresenceResponse { online:bool; }
`
	writeFile(t, schemaA, common+`
rpc_service Plugin {
  Update(UpdateRequest):UpdateResponse;
}

rpc_service Presence {
  Get(PresenceRequest):PresenceResponse;
}

rpc_service Rng {
  Int(IntRequest):IntResponse;
}
`)
	writeFile(t, schemaB, common+`
rpc_service Rng {
  Int(IntRequest):IntResponse;
}

rpc_service Plugin {
  Update(UpdateRequest):UpdateResponse;
}

rpc_service Presence {
  Get(PresenceRequest):PresenceResponse;
}
`)

	contractA := loadContractForTest(t, runner, schemaA, "modules")
	contractB := loadContractForTest(t, runner, schemaB, "modules")
	if contractA.SchemaHash != contractB.SchemaHash {
		t.Fatalf("schema hash should remain stable when host service declaration order changes")
	}
}

func loadContractForTest(
	t *testing.T,
	runner *flatc.Runner,
	schemaPath string,
	pkg string,
) Contract {
	t.Helper()

	bfbsPath, err := runner.GenerateBFBS(schemaPath, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("generate bfbs: %v", err)
	}
	model, err := Load(LoadOptions{
		SchemaPath:  schemaPath,
		BFBSPath:    bfbsPath,
		PackageName: pkg,
	})
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	return model
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
