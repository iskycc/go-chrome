//go:build integration

package db

import (
	"path/filepath"
	"testing"

	"go-chrome/internal/flow"
)

func TestIntegration_FlowSaveLoadRoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "integration.db")
	sqliteDB, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer sqliteDB.Close()

	store, err := NewFlowStore(sqliteDB)
	if err != nil {
		t.Fatalf("new flow store: %v", err)
	}

	f := flow.NewFlow("RoundTrip Flow")
	f.Description = "integration test flow"
	f.Tags = []string{"e2e", "integration"}

	sNavigate := flow.NewStep("open", flow.StepNavigate)
	sNavigate.Target = flow.Target{Strategy: flow.TargetXPath, Value: "http://localhost:18080"}
	f.Steps = append(f.Steps, sNavigate)

	sUser := flow.NewStep("user", flow.StepInput)
	sUser.Target = flow.Target{Strategy: flow.TargetXPath, Value: "//input[@id='username']"}
	sUser.Input = flow.Input{Mode: flow.InputTemplate, Text: "${var:u=SP${11000-11099}}"}
	f.Steps = append(f.Steps, sUser)

	sPass := flow.NewStep("pass", flow.StepInput)
	sPass.Target = flow.Target{Strategy: flow.TargetXPath, Value: "//input[@id='password']"}
	sPass.Input = flow.Input{Mode: flow.InputLiteral, Text: "Password123", MaskInLogs: true}
	f.Steps = append(f.Steps, sPass)

	sSubmit := flow.NewStep("submit", flow.StepClick)
	sSubmit.Target = flow.Target{Strategy: flow.TargetXPath, Value: `//button[@type='submit']`}
	f.Steps = append(f.Steps, sSubmit)

	sWelcome := flow.NewStep("welcome", flow.StepAssertExists)
	sWelcome.Target = flow.Target{Strategy: flow.TargetXPath, Value: `//div[contains(text(),'欢迎')]`}
	f.Steps = append(f.Steps, sWelcome)

	f.Steps = append(f.Steps, flow.NewStep("screenshot", flow.StepScreenshot))

	if err := store.Save(f); err != nil {
		t.Fatalf("save flow: %v", err)
	}

	loaded, err := store.Load(f.ID)
	if err != nil {
		t.Fatalf("load flow: %v", err)
	}

	if loaded.ID != f.ID {
		t.Fatalf("id mismatch: %s != %s", loaded.ID, f.ID)
	}
	if loaded.Name != f.Name {
		t.Fatalf("name mismatch: %s != %s", loaded.Name, f.Name)
	}
	if loaded.Description != f.Description {
		t.Fatalf("description mismatch: %s != %s", loaded.Description, f.Description)
	}
	if len(loaded.Tags) != len(f.Tags) || loaded.Tags[0] != f.Tags[0] || loaded.Tags[1] != f.Tags[1] {
		t.Fatalf("tags mismatch: %v != %v", loaded.Tags, f.Tags)
	}
	if len(loaded.Steps) != len(f.Steps) {
		t.Fatalf("step count mismatch: %d != %d", len(loaded.Steps), len(f.Steps))
	}

	for i := range f.Steps {
		orig := f.Steps[i]
		got := loaded.Steps[i]
		if got.Name != orig.Name {
			t.Errorf("step %d name mismatch: %s != %s", i, got.Name, orig.Name)
		}
		if got.Type != orig.Type {
			t.Errorf("step %d type mismatch: %s != %s", i, got.Type, orig.Type)
		}
		if got.Target.Value != orig.Target.Value {
			t.Errorf("step %d target value mismatch: %s != %s", i, got.Target.Value, orig.Target.Value)
		}
		if got.Input.Mode != orig.Input.Mode {
			t.Errorf("step %d input mode mismatch: %s != %s", i, got.Input.Mode, orig.Input.Mode)
		}
		if got.Input.Text != orig.Input.Text {
			t.Errorf("step %d input text mismatch: %s != %s", i, got.Input.Text, orig.Input.Text)
		}
		if got.Input.MaskInLogs != orig.Input.MaskInLogs {
			t.Errorf("step %d input mask mismatch: %v != %v", i, got.Input.MaskInLogs, orig.Input.MaskInLogs)
		}
		if got.OnError != orig.OnError {
			t.Errorf("step %d onError mismatch: %s != %s", i, got.OnError, orig.OnError)
		}
	}

	all, err := store.List()
	if err != nil {
		t.Fatalf("list flows: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(all))
	}

	exportPath := filepath.Join(t.TempDir(), "roundtrip-export.json")
	if err := store.Export(f.ID, exportPath); err != nil {
		t.Fatalf("export: %v", err)
	}

	imported, err := store.Import(exportPath)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if imported.ID == f.ID {
		t.Fatal("import should assign a new id")
	}
	if imported.Name != f.Name {
		t.Fatalf("imported name mismatch: %s != %s", imported.Name, f.Name)
	}
	if len(imported.Steps) != len(f.Steps) {
		t.Fatalf("imported step count mismatch: %d != %d", len(imported.Steps), len(f.Steps))
	}

	if err := store.Delete(imported.ID); err != nil {
		t.Fatalf("delete imported: %v", err)
	}
	if err := store.Delete(f.ID); err != nil {
		t.Fatalf("delete original: %v", err)
	}

	_, err = store.Load(f.ID)
	if err == nil {
		t.Fatal("expected error loading deleted flow")
	}
}
