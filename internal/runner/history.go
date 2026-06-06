package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// HistoryStore persists run results to disk.
type HistoryStore struct {
	dir string
}

// NewHistoryStore creates a new history store.
func NewHistoryStore(dir string) (*HistoryStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &HistoryStore{dir: dir}, nil
}

func (s *HistoryStore) path(flowID, runID string) string {
	return filepath.Join(s.dir, flowID, runID+".json")
}

// Save writes a run result.
func (s *HistoryStore) Save(result *RunResult) error {
	p := s.path(result.FlowID, result.StartedAt.Format("20060102-150405.000"))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// List returns history for a flow, sorted by startedAt desc.
func (s *HistoryStore) List(flowID string) ([]*RunResult, error) {
	p := filepath.Join(s.dir, flowID)
	entries, err := os.ReadDir(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var results []*RunResult
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p, e.Name()))
		if err != nil {
			continue
		}
		var r RunResult
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		results = append(results, &r)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt.After(results[j].StartedAt)
	})
	return results, nil
}

// ListAll returns all history entries across all flows.
func (s *HistoryStore) ListAll() ([]*RunResult, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var results []*RunResult
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		r, _ := s.List(e.Name())
		results = append(results, r...)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].StartedAt.After(results[j].StartedAt)
	})
	return results, nil
}

// Cleanup removes history older than retention days.
func (s *HistoryStore) Cleanup(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return filepath.Walk(s.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(path)
		}
		return nil
	})
}
