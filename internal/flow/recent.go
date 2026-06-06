package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RecentStore tracks recently opened flows.
type RecentStore struct {
	path   string
	FlowIDs []string `json:"flowIds"`
}

// NewRecentStore creates a new recent store.
func NewRecentStore(path string) (*RecentStore, error) {
	rs := &RecentStore{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return rs, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, rs); err != nil {
		return rs, nil // ignore corrupt file
	}
	return rs, nil
}

// Save persists the recent list.
func (rs *RecentStore) Save() error {
	if err := os.MkdirAll(filepath.Dir(rs.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rs.path, data, 0644)
}

// Touch moves a flow ID to the front of the recent list.
func (rs *RecentStore) Touch(flowID string) {
	// Remove existing
	var filtered []string
	for _, id := range rs.FlowIDs {
		if id != flowID {
			filtered = append(filtered, id)
		}
	}
	// Add to front
	rs.FlowIDs = append([]string{flowID}, filtered...)
	if len(rs.FlowIDs) > 20 {
		rs.FlowIDs = rs.FlowIDs[:20]
	}
}

// Remove removes a flow ID.
func (rs *RecentStore) Remove(flowID string) {
	var filtered []string
	for _, id := range rs.FlowIDs {
		if id != flowID {
			filtered = append(filtered, id)
		}
	}
	rs.FlowIDs = filtered
}
