package flow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	f := NewFlow("Test Flow")
	f.Description = "desc"
	f.Tags = []string{"tag1"}
	f.Steps = []Step{NewStep("Click", StepClick), NewStep("Input", StepInput)}

	if err := store.Save(f); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(f.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Name != f.Name {
		t.Fatalf("name mismatch")
	}
	if len(loaded.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(loaded.Steps))
	}

	flows, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}

	if err := store.Delete(f.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(f.ID); !os.IsNotExist(err) {
		t.Fatal("expected not found after delete")
	}
}

func TestStoreSearch(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)
	f1 := NewFlow("Login Flow")
	f1.Tags = []string{"auth"}
	store.Save(f1)
	f2 := NewFlow("Register Flow")
	store.Save(f2)

	results, _ := store.Search("login")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	results, _ = store.Search("auth")
	if len(results) != 1 {
		t.Fatalf("expected 1 tag result, got %d", len(results))
	}
}

func TestClone(t *testing.T) {
	f := NewFlow("Original")
	f.Steps = []Step{NewStep("A", StepClick)}
	cf := f.Clone()
	if cf.Name != "Original (Copy)" {
		t.Fatalf("unexpected clone name: %s", cf.Name)
	}
	if cf.ID == f.ID {
		t.Fatal("clone id should differ")
	}
	if len(cf.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(cf.Steps))
	}
	if cf.Steps[0].ID == f.Steps[0].ID {
		t.Fatal("clone step id should differ")
	}
}

func TestValidateFlow(t *testing.T) {
	f := NewFlow("Valid")
	f.Steps = []Step{NewStep("Click", StepClick)}
	f.Steps[0].Target.Value = "//button"
	if err := Validate(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f.Name = ""
	if err := Validate(f); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestImportExport(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)
	f := NewFlow("ExportMe")
	store.Save(f)

	exportPath := filepath.Join(dir, "export.json")
	if err := store.Export(f.ID, exportPath); err != nil {
		t.Fatal(err)
	}

	imported, err := store.Import(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Name != f.Name {
		t.Fatal("name mismatch after import")
	}
	if imported.ID == f.ID {
		t.Fatal("import should assign new id")
	}
}
