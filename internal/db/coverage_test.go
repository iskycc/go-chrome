package db

import (
	"os"
	"path/filepath"
	"testing"

	"go-chrome/internal/flow"
	"go-chrome/internal/runner"
)

func TestOpenErrorPaths(t *testing.T) {
	// Path where the parent is a file, so MkdirAll should fail.
	filePath := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if _, err := Open(filepath.Join(filePath, "test.db")); err == nil {
		t.Fatal("expected error when data dir cannot be created")
	}
}

func TestFlowStoreList(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := NewFlowStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	if _, err := store.List(); err != nil {
		t.Fatalf("list empty: %v", err)
	}

	f := flow.NewFlow("List Flow")
	if err := store.Save(f); err != nil {
		t.Fatalf("save: %v", err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one flow, got %d", len(list))
	}
}

func TestEnvRepoUniqueNameDuringImport(t *testing.T) {
	dir := t.TempDir()
	db1, err := Open(filepath.Join(dir, "source.db"))
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	source := NewEnvRepo(db1)
	if err := source.Save(&Environment{ID: "e1", Name: "默认环境"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	path := filepath.Join(dir, "env.json")
	if err := source.Export(path); err != nil {
		t.Fatalf("export: %v", err)
	}
	db1.Close()

	db2, err := Open(filepath.Join(dir, "target.db"))
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	defer db2.Close()
	target := NewEnvRepo(db2)
	// Pre-create a default env so Import doesn't create one itself.
	if err := target.Save(&Environment{ID: "default", Name: "默认环境"}); err != nil {
		t.Fatalf("save default: %v", err)
	}
	// First import renames to "默认环境 导入".
	if err := target.Import(path); err != nil {
		t.Fatalf("first import: %v", err)
	}
	// Second import should trigger uniqueEnvName suffix increment.
	if err := target.Import(path); err != nil {
		t.Fatalf("second import: %v", err)
	}
	envs, err := target.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(envs) != 3 {
		t.Fatalf("expected 3 envs, got %d: %+v", len(envs), envs)
	}
}

func TestEnvRepoClosedConnectionErrors(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	repo := NewEnvRepo(db)
	env := &Environment{ID: "env", Name: "Env"}
	if err := repo.Save(env); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Close the connection and verify subsequent operations return errors.
	db.Close()
	if _, err := repo.Get(env.ID); err == nil {
		t.Fatal("expected error after closing connection")
	}
	if _, err := repo.GetByName(env.Name); err == nil {
		t.Fatal("expected error after closing connection")
	}
	if _, err := repo.List(); err == nil {
		t.Fatal("expected error after closing connection")
	}
	if _, err := repo.GetActive(); err == nil {
		t.Fatal("expected error after closing connection")
	}
	if err := repo.SetActive(env.ID); err == nil {
		t.Fatal("expected error after closing connection")
	}
}

func TestApplyMigrationUnknownVersion(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := applyMigration(db.Conn, 999); err == nil {
		t.Fatal("expected error for unknown migration version")
	}
}

func TestFlowRepoSaveErrorPath(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewFlowRepo(db)
	// Drop the flows table to force Save to fail on insert.
	if _, err := db.Conn.Exec("DROP TABLE flows"); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	f := flow.NewFlow("Fail")
	if err := repo.Save(f); err == nil {
		t.Fatal("expected save error after dropping table")
	}
}

func TestDBErrorPathsAfterClose(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	flowRepo := NewFlowRepo(db)
	envRepo := NewEnvRepo(db)
	store, _ := NewFlowStore(db)
	runRepo := NewRunRepo(db)

	f := flow.NewFlow("Flow")
	_ = flowRepo.Save(f)
	env := &Environment{ID: "env", Name: "Env"}
	_ = envRepo.Save(env)

	db.Close()

	mustErr := func(label string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("expected error for %s after close", label)
		}
	}

	// FlowRepo
	mustErr("flowRepo.Save", flowRepo.Save(f))
	mustErr("flowRepo.Get", func() error { _, err := flowRepo.Get(f.ID); return err }())
	mustErr("flowRepo.List", func() error { _, err := flowRepo.List(); return err }())
	mustErr("flowRepo.ListSorted", func() error { _, err := flowRepo.ListSorted(); return err }())
	mustErr("flowRepo.Delete", flowRepo.Delete(f.ID))
	mustErr("flowRepo.Search", func() error { _, err := flowRepo.Search("x", ""); return err }())

	// EnvRepo
	mustErr("envRepo.Save", envRepo.Save(env))
	mustErr("envRepo.Get", func() error { _, err := envRepo.Get(env.ID); return err }())
	mustErr("envRepo.GetByName", func() error { _, err := envRepo.GetByName(env.Name); return err }())
	mustErr("envRepo.List", func() error { _, err := envRepo.List(); return err }())
	mustErr("envRepo.Delete", envRepo.Delete(env.ID))
	mustErr("envRepo.SetActive", envRepo.SetActive(env.ID))
	mustErr("envRepo.SaveVar", envRepo.SaveVar(&EnvironmentVariable{ID: "v", EnvironmentID: env.ID, Key: "K", Value: "v"}))
	mustErr("envRepo.ListVars", func() error { _, err := envRepo.ListVars(env.ID); return err }())
	mustErr("envRepo.GetVar", func() error { _, err := envRepo.GetVar(env.ID, "K"); return err }())
	mustErr("envRepo.DeleteVar", envRepo.DeleteVar("v"))
	mustErr("envRepo.Export", envRepo.Export(filepath.Join(dir, "env.json")))

	// FlowStore
	mustErr("store.List", func() error { _, err := store.List(); return err }())
	mustErr("store.ListSorted", func() error { _, err := store.ListSorted(); return err }())
	mustErr("store.Export", store.Export(f.ID, filepath.Join(dir, "flow.json")))

	// RunRepo
	mustErr("runRepo.Save", runRepo.Save(&runner.RunResult{ID: "r", FlowID: f.ID}, env.ID))
	mustErr("runRepo.ListByFlow", func() error { _, err := runRepo.ListByFlow(f.ID, 10); return err }())
	mustErr("runRepo.ListAll", func() error { _, err := runRepo.ListAll(10); return err }())
	mustErr("runRepo.ListFiltered", func() error { _, err := runRepo.ListFiltered("", "", runner.StatusSuccess, 10); return err }())
	mustErr("runRepo.Cleanup", runRepo.Cleanup(1))
}

func TestEnvRepoCreateDefaultIfNoneExisting(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewEnvRepo(db)
	if err := repo.CreateDefaultIfNone(); err != nil {
		t.Fatalf("first create default: %v", err)
	}
	// Second call should be a no-op.
	if err := repo.CreateDefaultIfNone(); err != nil {
		t.Fatalf("second create default: %v", err)
	}
	envs, err := repo.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected one env, got %d", len(envs))
	}
}

func TestEnvProviderMissingVar(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewEnvRepo(db)
	env := &Environment{ID: "env", Name: "Env"}
	if err := repo.Save(env); err != nil {
		t.Fatalf("save: %v", err)
	}
	provider := repo.EnvProvider(env.ID)
	_, found, _ := provider.GetEnvValue("MISSING")
	if found {
		t.Fatal("expected missing variable")
	}
}

