package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-chrome/internal/flow"
	"go-chrome/internal/runner"
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

func TestSQLiteFlowStoreImportExportAndSearch(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := NewFlowStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	f := flow.NewFlow("Login Flow")
	f.Description = "auth flow"
	f.Tags = []string{"auth"}
	f.Steps = []flow.Step{flow.NewStep("Open", flow.StepNavigate)}
	f.Steps[0].Target.Value = "https://example.test"
	if err := store.Save(f); err != nil {
		t.Fatalf("save: %v", err)
	}

	results, err := store.Search("auth")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one search result, got %d", len(results))
	}

	exportPath := t.TempDir() + "/flow.json"
	if err := store.Export(f.ID, exportPath); err != nil {
		t.Fatalf("export: %v", err)
	}
	imported, err := store.Import(exportPath)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if imported.ID == f.ID {
		t.Fatal("import should assign a new ID")
	}
	if err := store.Delete(f.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestSQLiteFlowStoreListSortedLoadAndImportErrors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := NewFlowStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	oldFlow := flow.NewFlow("Old")
	newFlow := flow.NewFlow("New")
	if err := store.Save(oldFlow); err != nil {
		t.Fatalf("save old: %v", err)
	}
	if err := store.Save(newFlow); err != nil {
		t.Fatalf("save new: %v", err)
	}
	oldUpdated := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	newUpdated := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Conn.Exec("UPDATE flows SET updated_at = ? WHERE id = ?", oldUpdated, oldFlow.ID); err != nil {
		t.Fatalf("set old updated_at: %v", err)
	}
	if _, err := db.Conn.Exec("UPDATE flows SET updated_at = ? WHERE id = ?", newUpdated, newFlow.ID); err != nil {
		t.Fatalf("set new updated_at: %v", err)
	}
	sorted, err := store.ListSorted()
	if err != nil {
		t.Fatalf("list sorted: %v", err)
	}
	if len(sorted) != 2 || sorted[0].ID != newFlow.ID {
		t.Fatalf("expected newest flow first, got %+v", sorted)
	}
	loaded, err := store.Load(oldFlow.ID)
	if err != nil {
		t.Fatalf("load flow: %v", err)
	}
	if loaded.Name != oldFlow.Name {
		t.Fatalf("unexpected loaded flow: %+v", loaded)
	}

	if _, err := store.Import(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("expected missing import file error")
	}
	badJSON := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badJSON, []byte("{bad"), 0644); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	if _, err := store.Import(badJSON); err == nil {
		t.Fatal("expected invalid json import error")
	}
	invalidFlow := filepath.Join(dir, "invalid-flow.json")
	if err := os.WriteFile(invalidFlow, []byte(`{"id":"","name":""}`), 0644); err != nil {
		t.Fatalf("write invalid flow: %v", err)
	}
	if _, err := store.Import(invalidFlow); err == nil {
		t.Fatal("expected invalid flow import error")
	}
	if err := store.Export("missing-flow", filepath.Join(dir, "missing-export.json")); err == nil {
		t.Fatal("expected missing flow export error")
	}
}

func TestFlowRepoListSortedAndSearchFilters(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewFlowRepo(db)
	flows := []*flow.Flow{
		{ID: "b", Name: "Beta", Description: "payment", Tags: []string{"billing"}},
		{ID: "a", Name: "alpha", Description: "auth", Tags: []string{"login"}},
	}
	for _, f := range flows {
		if err := repo.Save(f); err != nil {
			t.Fatalf("save %s: %v", f.ID, err)
		}
	}
	sorted, err := repo.ListSorted()
	if err != nil {
		t.Fatalf("list sorted: %v", err)
	}
	if len(sorted) != 2 || sorted[0].Name != "alpha" {
		t.Fatalf("expected alpha first, got %+v", sorted)
	}
	byTag, err := repo.Search("", "billing")
	if err != nil {
		t.Fatalf("search tag: %v", err)
	}
	if len(byTag) != 1 || byTag[0].ID != "b" {
		t.Fatalf("unexpected tag results: %+v", byTag)
	}
	noMatch, err := repo.Search("missing", "")
	if err != nil {
		t.Fatalf("search missing: %v", err)
	}
	if len(noMatch) != 0 {
		t.Fatalf("expected no matches, got %+v", noMatch)
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

func TestEnvRepoVarsAndActiveEnvironment(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewEnvRepo(db)
	envA := &Environment{ID: "env-a", Name: "A", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	envB := &Environment{ID: "env-b", Name: "B", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Save(envA); err != nil {
		t.Fatalf("save env a: %v", err)
	}
	if err := repo.Save(envB); err != nil {
		t.Fatalf("save env b: %v", err)
	}
	if err := repo.SetActive(envB.ID); err != nil {
		t.Fatalf("set active: %v", err)
	}
	active, err := repo.GetActive()
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active.ID != envB.ID {
		t.Fatalf("active env mismatch: %s", active.ID)
	}

	v := &EnvironmentVariable{
		ID:            "var-1",
		EnvironmentID: envB.ID,
		Key:           "PASSWORD",
		Value:         "secret-value",
		IsSecret:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := repo.SaveVar(v); err != nil {
		t.Fatalf("save var: %v", err)
	}
	got, err := repo.GetVar(envB.ID, "PASSWORD")
	if err != nil {
		t.Fatalf("get var: %v", err)
	}
	if !got.IsSecret || got.Value != "secret-value" {
		t.Fatalf("unexpected var: %+v", got)
	}

	provider := repo.EnvProvider(envB.ID)
	value, found, secret := provider.GetEnvValue("PASSWORD")
	if !found || !secret || value != "secret-value" {
		t.Fatalf("unexpected provider value: value=%q found=%v secret=%v", value, found, secret)
	}

	if err := repo.DeleteVar(v.ID); err != nil {
		t.Fatalf("delete var: %v", err)
	}
	if _, err := repo.GetVar(envB.ID, "PASSWORD"); err == nil {
		t.Fatal("expected missing var after delete")
	}
	if err := repo.Delete(envA.ID); err != nil {
		t.Fatalf("delete env: %v", err)
	}
}

func TestEnvRepoMissingAndListVars(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewEnvRepo(db)
	if _, err := repo.Get("missing"); err == nil {
		t.Fatal("expected missing env by id")
	}
	if _, err := repo.GetByName("missing"); err == nil {
		t.Fatal("expected missing env by name")
	}
	if _, err := repo.GetActive(); err == nil {
		t.Fatal("expected missing active env")
	}
	env := &Environment{ID: "env", Name: "Env"}
	if err := repo.Save(env); err != nil {
		t.Fatalf("save env: %v", err)
	}
	for _, key := range []string{"B", "A"} {
		if err := repo.SaveVar(&EnvironmentVariable{ID: "var-" + key, EnvironmentID: env.ID, Key: key, Value: key}); err != nil {
			t.Fatalf("save var %s: %v", key, err)
		}
	}
	vars, err := repo.ListVars(env.ID)
	if err != nil {
		t.Fatalf("list vars: %v", err)
	}
	if len(vars) != 2 || vars[0].Key != "A" || vars[1].Key != "B" {
		t.Fatalf("expected vars ordered by key, got %+v", vars)
	}
}

func TestEnvRepoImportExport(t *testing.T) {
	dir := t.TempDir()
	db1, err := Open(dir + "/source.db")
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	defer db1.Close()

	source := NewEnvRepo(db1)
	env := &Environment{ID: "env-1", Name: "测试环境", Description: "desc", IsActive: true}
	if err := source.Save(env); err != nil {
		t.Fatalf("save env: %v", err)
	}
	if err := source.SaveVar(&EnvironmentVariable{
		ID:            "var-1",
		EnvironmentID: env.ID,
		Key:           "PASSWORD",
		Value:         "secret-value",
		IsSecret:      true,
		Description:   "password",
	}); err != nil {
		t.Fatalf("save var: %v", err)
	}

	path := dir + "/env.json"
	if err := source.Export(path); err != nil {
		t.Fatalf("export env: %v", err)
	}

	db2, err := Open(dir + "/target.db")
	if err != nil {
		t.Fatalf("open target db: %v", err)
	}
	defer db2.Close()
	target := NewEnvRepo(db2)
	if err := target.Import(path); err != nil {
		t.Fatalf("import env: %v", err)
	}
	imported, err := target.GetByName("测试环境")
	if err != nil {
		t.Fatalf("get imported env: %v", err)
	}
	if !imported.IsActive {
		t.Fatal("expected imported env to be active")
	}
	importedVar, err := target.GetVar(imported.ID, "PASSWORD")
	if err != nil {
		t.Fatalf("get imported var: %v", err)
	}
	if importedVar.Value != "secret-value" || !importedVar.IsSecret {
		t.Fatalf("unexpected imported var: %+v", importedVar)
	}
}

func TestEnvRepoImportSkipsEmptyEntriesAndCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	path := filepath.Join(dir, "env.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"environments":[{"name":"","variables":[{"key":"IGNORED","value":"x"}]}]}`), 0644); err != nil {
		t.Fatalf("write env export: %v", err)
	}
	repo := NewEnvRepo(db)
	if err := repo.Import(path); err != nil {
		t.Fatalf("import: %v", err)
	}
	active, err := repo.GetActive()
	if err != nil {
		t.Fatalf("expected default active env: %v", err)
	}
	if active.Name != "默认环境" {
		t.Fatalf("unexpected default env: %+v", active)
	}
	if err := repo.Import(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("expected missing import file error")
	}
	badJSON := filepath.Join(dir, "bad-env.json")
	if err := os.WriteFile(badJSON, []byte("{bad"), 0644); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	if err := repo.Import(badJSON); err == nil {
		t.Fatal("expected bad json import error")
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

func TestRecentRepoEmptyLoadAndSave(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewRecentRepo(db)
	ids, err := repo.Load()
	if err != nil {
		t.Fatalf("load empty: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty recent ids, got %v", ids)
	}
	if err := repo.Save(nil); err != nil {
		t.Fatalf("save nil: %v", err)
	}
	ids, err = repo.Load()
	if err != nil {
		t.Fatalf("load saved empty: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty saved recent ids, got %v", ids)
	}
}

func TestRunRepoSaveAndListByFlow(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	flowRepo := NewFlowRepo(db)
	f := &flow.Flow{ID: "flow-1", Name: "Test Flow", Steps: []flow.Step{
		{ID: "step-1", Name: "Step 1", Enabled: true, Type: flow.StepInput},
	}}
	if err := flowRepo.Save(f); err != nil {
		t.Fatalf("save flow: %v", err)
	}

	envRepo := NewEnvRepo(db)
	env := &Environment{ID: "env-1", Name: "测试环境", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := envRepo.Save(env); err != nil {
		t.Fatalf("save env: %v", err)
	}

	repo := NewRunRepo(db)
	started := time.Now().Add(-time.Second)
	res := &runner.RunResult{
		ID:            "run-1",
		FlowID:        "flow-1",
		FlowName:      "Test Flow",
		EnvironmentID: "env-1",
		StartedAt:     started,
		FinishedAt:    time.Now(),
		Status:        runner.StatusSuccess,
		TotalSteps:    1,
		SuccessCount:  1,
		Steps: []runner.StepResult{
			{
				StepID:               "step-1",
				StepName:             "Step 1",
				Type:                 string(flow.StepInput),
				Status:               runner.StatusSuccess,
				GeneratedInputMasked: "abcd****wxyz",
			},
		},
	}
	if err := repo.Save(res, res.EnvironmentID); err != nil {
		t.Fatalf("save run: %v", err)
	}

	results, err := repo.ListByFlow("flow-1", 10)
	if err != nil {
		t.Fatalf("list by flow: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	got := results[0]
	if got.EnvironmentID != "env-1" {
		t.Fatalf("environment id mismatch: %q", got.EnvironmentID)
	}
	if len(got.Steps) != 1 {
		t.Fatalf("expected one step result, got %d", len(got.Steps))
	}
	if got.Steps[0].GeneratedInputMasked != "abcd****wxyz" {
		t.Fatalf("masked input mismatch: %q", got.Steps[0].GeneratedInputMasked)
	}
}

func TestRunRepoAllowsSameStepAcrossRuns(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	flowRepo := NewFlowRepo(db)
	f := &flow.Flow{ID: "flow-1", Name: "Test Flow", Steps: []flow.Step{
		{ID: "step-1", Name: "Step 1", Enabled: true, Type: flow.StepClick},
	}}
	if err := flowRepo.Save(f); err != nil {
		t.Fatalf("save flow: %v", err)
	}

	repo := NewRunRepo(db)
	for _, id := range []string{"run-1", "run-2"} {
		res := &runner.RunResult{
			ID:           id,
			FlowID:       "flow-1",
			StartedAt:    time.Now().Add(-time.Second),
			FinishedAt:   time.Now(),
			Status:       runner.StatusSuccess,
			TotalSteps:   1,
			SuccessCount: 1,
			Steps: []runner.StepResult{{
				StepID:   "step-1",
				StepName: "Step 1",
				Type:     string(flow.StepClick),
				Status:   runner.StatusSuccess,
			}},
		}
		if err := repo.Save(res, ""); err != nil {
			t.Fatalf("save %s: %v", id, err)
		}
	}
}

func TestRunRepoListAllAndCleanup(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	flowRepo := NewFlowRepo(db)
	f := &flow.Flow{ID: "flow-1", Name: "Test Flow"}
	if err := flowRepo.Save(f); err != nil {
		t.Fatalf("save flow: %v", err)
	}

	repo := NewRunRepo(db)
	old := &runner.RunResult{
		ID:         "old-run",
		FlowID:     "flow-1",
		StartedAt:  time.Now().AddDate(0, 0, -10),
		FinishedAt: time.Now().AddDate(0, 0, -10).Add(time.Second),
		Status:     runner.StatusFailed,
	}
	recent := &runner.RunResult{
		ID:         "recent-run",
		FlowID:     "flow-1",
		StartedAt:  time.Now(),
		FinishedAt: time.Now().Add(time.Second),
		Status:     runner.StatusSuccess,
	}
	if err := repo.Save(old, ""); err != nil {
		t.Fatalf("save old: %v", err)
	}
	if err := repo.Save(recent, ""); err != nil {
		t.Fatalf("save recent: %v", err)
	}

	all, err := repo.ListAll(10)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected two results, got %d", len(all))
	}
	if err := repo.Cleanup(7); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	all, err = repo.ListAll(10)
	if err != nil {
		t.Fatalf("list all after cleanup: %v", err)
	}
	if len(all) != 1 || all[0].ID != "recent-run" {
		t.Fatalf("expected only recent run, got %+v", all)
	}
}

func TestRunRepoListFiltered(t *testing.T) {
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	flowRepo := NewFlowRepo(db)
	for _, id := range []string{"flow-1", "flow-2"} {
		if err := flowRepo.Save(&flow.Flow{ID: id, Name: id}); err != nil {
			t.Fatalf("save flow %s: %v", id, err)
		}
	}
	envRepo := NewEnvRepo(db)
	for _, id := range []string{"env-1", "env-2"} {
		if err := envRepo.Save(&Environment{ID: id, Name: id}); err != nil {
			t.Fatalf("save env %s: %v", id, err)
		}
	}

	repo := NewRunRepo(db)
	runs := []struct {
		id     string
		flowID string
		envID  string
		status runner.Status
	}{
		{"run-1", "flow-1", "env-1", runner.StatusSuccess},
		{"run-2", "flow-1", "env-2", runner.StatusFailed},
		{"run-3", "flow-2", "env-1", runner.StatusSuccess},
	}
	for _, run := range runs {
		res := &runner.RunResult{
			ID:         run.id,
			FlowID:     run.flowID,
			StartedAt:  time.Now(),
			FinishedAt: time.Now().Add(time.Second),
			Status:     run.status,
		}
		if err := repo.Save(res, run.envID); err != nil {
			t.Fatalf("save %s: %v", run.id, err)
		}
	}

	results, err := repo.ListFiltered("flow-1", "env-1", runner.StatusSuccess, 10)
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if len(results) != 1 || results[0].ID != "run-1" {
		t.Fatalf("unexpected filtered results: %+v", results)
	}

	results, err = repo.ListFiltered("", "env-1", runner.StatusSuccess, 10)
	if err != nil {
		t.Fatalf("list filtered by env/status: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two env/status results, got %+v", results)
	}
}
