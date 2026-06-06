package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistorySaveAndList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewHistoryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	result := &RunResult{
		FlowID:     "flow-1",
		FlowName:   "Test Flow",
		StartedAt:  time.Now(),
		FinishedAt: time.Now(),
		Status:     StatusSuccess,
		Steps: []StepResult{
			{StepID: "s1", StepName: "Click", Status: StatusSuccess},
		},
	}
	if err := store.Save(result); err != nil {
		t.Fatalf("save: %v", err)
	}

	results, err := store.List("flow-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].FlowName != "Test Flow" {
		t.Fatalf("unexpected flow name: %s", results[0].FlowName)
	}
}

func TestHistoryListAll(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewHistoryStore(dir)
	store.Save(&RunResult{FlowID: "f1", StartedAt: time.Now(), Status: StatusSuccess})
	store.Save(&RunResult{FlowID: "f2", StartedAt: time.Now().Add(-time.Hour), Status: StatusFailed})

	all, err := store.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 results, got %d", len(all))
	}
}

func TestHistoryCleanup(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewHistoryStore(dir)
	oldTime := time.Now().AddDate(0, 0, -30)
	old := &RunResult{FlowID: "f1", StartedAt: oldTime, Status: StatusSuccess}
	store.Save(old)
	newTime := time.Now()
	new := &RunResult{FlowID: "f1", StartedAt: newTime, Status: StatusSuccess}
	store.Save(new)

	// Manually adjust file modification times to simulate age
	oldPath := filepath.Join(dir, "f1", oldTime.Format("20060102-150405.000")+".json")
	os.Chtimes(oldPath, oldTime, oldTime)

	if err := store.Cleanup(7); err != nil {
		t.Fatal(err)
	}
	results, _ := store.List("f1")
	if len(results) != 1 {
		t.Fatalf("expected 1 result after cleanup, got %d", len(results))
	}
}
