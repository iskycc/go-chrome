package flow

import "testing"

func TestMigrateZeroToCurrent(t *testing.T) {
	f := &Flow{
		SchemaVersion: 0,
		ID:            "flow-1",
		Name:          "Legacy",
		Steps: []Step{{
			ID:      "step-1",
			Name:    "Click",
			Enabled: true,
			Type:    StepClick,
			Target:  Target{Strategy: TargetXPath, Value: "//button"},
		}},
	}

	if err := Migrate(f); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if f.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schema version mismatch: %d", f.SchemaVersion)
	}
	if f.Steps[0].OnError != ErrStop {
		t.Fatalf("expected default onError stop, got %s", f.Steps[0].OnError)
	}
	if f.Steps[0].TimeoutMs != 10000 {
		t.Fatalf("expected default timeout, got %d", f.Steps[0].TimeoutMs)
	}
}

func TestMigrateUnknownVersionTreatsAsCurrent(t *testing.T) {
	f := &Flow{SchemaVersion: -1, ID: "flow-1", Name: "Unknown"}
	if err := Migrate(f); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if f.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("expected current schema, got %d", f.SchemaVersion)
	}
}
