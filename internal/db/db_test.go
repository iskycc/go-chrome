package db

import (
	"testing"

	"go-chrome/internal/flow"
)

func TestOpenAndMigrate(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Verify tables exist
	tables := []string{
		"schema_migrations", "app_state", "flows", "steps",
		"environments", "environment_variables",
		"run_results", "run_step_results",
	}
	for _, table := range tables {
		var name string
		err := db.Conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestRepeatMigrate(t *testing.T) {
	path := t.TempDir() + "/test.db"
	db1, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.Close()

	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	db2.Close()
}

func TestFlowRepoCRUD(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewFlowRepo(db)
	f := &flow.Flow{ID: "flow-1", Name: "Test Flow", Steps: []flow.Step{
		{ID: "step-1", Name: "Step 1", Type: flow.StepClick},
	}}
	if err := repo.Save(f); err != nil {
		t.Fatalf("save flow: %v", err)
	}

	loaded, err := repo.Get("flow-1")
	if err != nil {
		t.Fatalf("get flow: %v", err)
	}
	if loaded.Name != "Test Flow" {
		t.Errorf("name mismatch: got %q", loaded.Name)
	}
	if len(loaded.Steps) != 1 {
		t.Errorf("steps mismatch: got %d", len(loaded.Steps))
	}

	if err := repo.Delete("flow-1"); err != nil {
		t.Fatalf("delete flow: %v", err)
	}
	_, err = repo.Get("flow-1")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestEnvRepo(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewEnvRepo(db)
	if err := repo.CreateDefaultIfNone(); err != nil {
		t.Fatalf("create default: %v", err)
	}

	envs, err := repo.List()
	if err != nil {
		t.Fatalf("list envs: %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 default env, got %d", len(envs))
	}

	active, err := repo.GetActive()
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active.Name != "默认环境" {
		t.Errorf("unexpected active env: %q", active.Name)
	}
}

func TestRecentRepo(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewRecentRepo(db)
	if err := repo.Save([]string{"a", "b", "c"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	ids, err := repo.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(ids) != 3 || ids[0] != "a" {
		t.Errorf("unexpected ids: %v", ids)
	}
}
