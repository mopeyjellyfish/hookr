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
table LookupRequest { key:string; }
table LookupResponse { ok:bool; }

rpc_service Lookup {
  Get(LookupRequest):LookupResponse;
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
	if model.HostServices[0].Name != "Lookup" || model.HostServices[1].Name != "Rng" {
		t.Fatalf("host services = %#v, want Lookup,Rng", model.HostServices)
	}
	if _, ok := model.HostMethod("Rng", "Int"); !ok {
		t.Fatal("expected Rng.Int host method")
	}
	if _, ok := model.HostMethod("Lookup", "Get"); !ok {
		t.Fatal("expected Lookup.Get host method")
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
table LookupRequest { key:string; }
table LookupResponse { ok:bool; }
`
	writeFile(t, schemaA, common+`
rpc_service Plugin {
  Update(UpdateRequest):UpdateResponse;
}

rpc_service Lookup {
  Get(LookupRequest):LookupResponse;
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

rpc_service Lookup {
  Get(LookupRequest):LookupResponse;
}
`)

	contractA := loadContractForTest(t, runner, schemaA, "modules")
	contractB := loadContractForTest(t, runner, schemaB, "modules")
	if contractA.SchemaHash != contractB.SchemaHash {
		t.Fatalf("schema hash should remain stable when host service declaration order changes")
	}
}

func TestLoadContractResolvesQualifiedPluginService(t *testing.T) {
	t.Parallel()

	runner, err := flatc.New("")
	if err != nil {
		t.Skipf("flatc not available: %v", err)
	}

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "qualified.fbs")
	writeFile(t, schemaPath, `
namespace Hookr.Test.Modules;
table UpdateRequest {}
table UpdateResponse { ok:bool; }
table LookupRequest {}
table LookupResponse { ok:bool; }

namespace Hookr;
rpc_service Plugin {
  Update(Hookr.Test.Modules.UpdateRequest):Hookr.Test.Modules.UpdateResponse;
}

namespace Hookr.Host;
rpc_service Lookup {
  Get(Hookr.Test.Modules.LookupRequest):Hookr.Test.Modules.LookupResponse;
}
`)

	bfbsPath, err := runner.GenerateBFBS(schemaPath, tmpDir, nil)
	if err != nil {
		t.Fatalf("generate bfbs: %v", err)
	}
	model, err := Load(LoadOptions{
		SchemaPath:        schemaPath,
		BFBSPath:          bfbsPath,
		PackageName:       "qualified",
		PluginServiceName: "Hookr.Plugin",
	})
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	if model.PluginService.Name != "Plugin" {
		t.Fatalf("plugin service = %q, want Plugin short name", model.PluginService.Name)
	}
	if got := len(model.HostServices); got != 1 {
		t.Fatalf("host services = %d, want 1", got)
	}
	if model.HostServices[0].Name != "Lookup" {
		t.Fatalf("host service short name = %q, want Lookup", model.HostServices[0].Name)
	}
}

func TestLoadContractRejectsAmbiguousShortPluginServiceName(t *testing.T) {
	t.Parallel()

	runner, err := flatc.New("")
	if err != nil {
		t.Skipf("flatc not available: %v", err)
	}

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "ambiguous.fbs")
	writeFile(t, schemaPath, `
namespace Hookr.Test.Modules;
table UpdateRequest {}
table UpdateResponse { ok:bool; }

namespace Hookr;
rpc_service Plugin {
  Update(Hookr.Test.Modules.UpdateRequest):Hookr.Test.Modules.UpdateResponse;
}

namespace Example;
rpc_service Plugin {
  Update(Hookr.Test.Modules.UpdateRequest):Hookr.Test.Modules.UpdateResponse;
}
`)

	bfbsPath, err := runner.GenerateBFBS(schemaPath, tmpDir, nil)
	if err != nil {
		t.Fatalf("generate bfbs: %v", err)
	}
	_, err = Load(LoadOptions{
		SchemaPath:        schemaPath,
		BFBSPath:          bfbsPath,
		PackageName:       "ambiguous",
		PluginServiceName: "Plugin",
	})
	if err == nil {
		t.Fatal("expected ambiguous plugin service error")
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
