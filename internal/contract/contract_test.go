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
