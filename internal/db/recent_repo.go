package db

import (
	"strings"
	"time"
)

// RecentRepo tracks recently opened flows in SQLite app_state.
type RecentRepo struct {
	db *DB
}

// NewRecentRepo creates a recent repo.
func NewRecentRepo(db *DB) *RecentRepo {
	return &RecentRepo{db: db}
}

// Load returns the recent flow IDs.
func (r *RecentRepo) Load() ([]string, error) {
	var value string
	row := r.db.Conn.QueryRow("SELECT value FROM app_state WHERE key = 'recent_flows'")
	if err := row.Scan(&value); err != nil {
		return nil, nil // not found is ok
	}
	if value == "" {
		return nil, nil
	}
	return strings.Split(value, ","), nil
}

// Save persists the recent flow IDs.
func (r *RecentRepo) Save(flowIDs []string) error {
	value := strings.Join(flowIDs, ",")
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Conn.Exec(`
		INSERT INTO app_state (key, value, updated_at) VALUES ('recent_flows', ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, value, now)
	return err
}
