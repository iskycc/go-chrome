package db

import (
	"encoding/json"
	"os"
	"sort"
	"time"

	"github.com/google/uuid"
	"go-chrome/internal/flow"
)

// FlowStore implements the old flow.Store interface backed by SQLite.
type FlowStore struct {
	repo *FlowRepo
}

// NewFlowStore creates a flow store backed by SQLite.
func NewFlowStore(sqliteDB *DB) (*FlowStore, error) {
	return &FlowStore{repo: NewFlowRepo(sqliteDB)}, nil
}

// Save writes a flow to SQLite.
func (s *FlowStore) Save(f *flow.Flow) error {
	f.UpdatedAt = time.Now()
	return s.repo.Save(f)
}

// Load reads a flow by ID.
func (s *FlowStore) Load(id string) (*flow.Flow, error) {
	return s.repo.Get(id)
}

// Delete removes a flow.
func (s *FlowStore) Delete(id string) error {
	return s.repo.Delete(id)
}

// List returns all flows.
func (s *FlowStore) List() ([]*flow.Flow, error) {
	return s.repo.List()
}

// ListSorted returns flows sorted by updatedAt desc.
func (s *FlowStore) ListSorted() ([]*flow.Flow, error) {
	flows, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(flows, func(i, j int) bool {
		return flows[i].UpdatedAt.After(flows[j].UpdatedAt)
	})
	return flows, nil
}

// Search searches flows by name or tag.
func (s *FlowStore) Search(query string) ([]*flow.Flow, error) {
	return s.repo.Search(query, "")
}

// Import imports a flow JSON file into SQLite, assigning a new ID.
func (s *FlowStore) Import(path string) (*flow.Flow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f flow.Flow
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if err := flow.Validate(&f); err != nil {
		return nil, err
	}
	f.ID = uuid.New().String()
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
	for i := range f.Steps {
		f.Steps[i].ID = uuid.New().String()
	}
	if err := s.Save(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Export writes a flow to a custom JSON path.
func (s *FlowStore) Export(id string, path string) error {
	f, err := s.Load(id)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
