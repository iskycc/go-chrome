package flow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Store handles flow persistence.
type Store struct {
	dir string
}

// NewStore creates a new flow store.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

// Save writes a flow to disk.
func (s *Store) Save(f *Flow) error {
	f.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(f.ID), data, 0644)
}

// Load reads a flow by ID.
func (s *Store) Load(id string) (*Flow, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var f Flow
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse flow %s: %w", id, err)
	}
	if f.SchemaVersion < CurrentSchemaVersion {
		if err := Migrate(&f); err != nil {
			return nil, fmt.Errorf("migrate flow %s: %w", id, err)
		}
	}
	return &f, nil
}

// Delete removes a flow.
func (s *Store) Delete(id string) error {
	return os.Remove(s.path(id))
}

// List returns all flows.
func (s *Store) List() ([]*Flow, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var flows []*Flow
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		f, err := s.Load(id)
		if err != nil {
			continue // skip corrupted
		}
		flows = append(flows, f)
	}
	return flows, nil
}

// ListSorted returns flows sorted by updatedAt desc.
func (s *Store) ListSorted() ([]*Flow, error) {
	flows, err := s.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(flows, func(i, j int) bool {
		return flows[i].UpdatedAt.After(flows[j].UpdatedAt)
	})
	return flows, nil
}

// Search searches flows by name or tag.
func (s *Store) Search(query string) ([]*Flow, error) {
	flows, err := s.List()
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	var results []*Flow
	for _, f := range flows {
		if strings.Contains(strings.ToLower(f.Name), query) ||
			strings.Contains(strings.ToLower(f.Description), query) {
			results = append(results, f)
			continue
		}
		for _, t := range f.Tags {
			if strings.Contains(strings.ToLower(t), query) {
				results = append(results, f)
				break
			}
		}
	}
	return results, nil
}

// Import imports a flow file, assigning a new ID.
func (s *Store) Import(path string) (*Flow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f Flow
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if err := Validate(&f); err != nil {
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

// Export writes a flow to a custom path.
func (s *Store) Export(id string, path string) error {
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
