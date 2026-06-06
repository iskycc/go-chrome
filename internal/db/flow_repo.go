package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go-chrome/internal/flow"
)

// FlowRepo handles flow persistence.
type FlowRepo struct {
	db *DB
}

// NewFlowRepo creates a flow repository.
func NewFlowRepo(db *DB) *FlowRepo {
	return &FlowRepo{db: db}
}

// Save inserts or updates a flow and its steps.
func (r *FlowRepo) Save(f *flow.Flow) error {
	tx, err := r.db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	tagsJSON, _ := json.Marshal(f.Tags)

	_, err = tx.Exec(`
		INSERT INTO flows (id, name, description, tags_json, schema_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			tags_json = excluded.tags_json,
			schema_version = excluded.schema_version,
			updated_at = excluded.updated_at
	`, f.ID, f.Name, f.Description, string(tagsJSON), f.SchemaVersion, now, now)
	if err != nil {
		return fmt.Errorf("save flow: %w", err)
	}

	// Delete old steps and re-insert to maintain order
	_, err = tx.Exec("DELETE FROM steps WHERE flow_id = ?", f.ID)
	if err != nil {
		return fmt.Errorf("delete old steps: %w", err)
	}

	for i, s := range f.Steps {
		if s.ID == "" {
			s.ID = uuid.New().String()
		}
		_, err = tx.Exec(`
			INSERT INTO steps (id, flow_id, sort_order, name, enabled, type, target_strategy, target_value,
				input_mode, input_text, input_mask_in_logs, wait_before_ms, wait_after_ms, timeout_ms,
				on_error, note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.ID, f.ID, i, s.Name, boolToInt(s.Enabled), string(s.Type), string(s.Target.Strategy),
			s.Target.Value, string(s.Input.Mode), s.Input.Text, boolToInt(s.Input.MaskInLogs),
			s.WaitBeforeMs, s.WaitAfterMs, s.TimeoutMs, string(s.OnError), s.Note, now, now)
		if err != nil {
			return fmt.Errorf("save step %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// Get loads a flow with its steps by ID.
func (r *FlowRepo) Get(id string) (*flow.Flow, error) {
	row := r.db.Conn.QueryRow(`
		SELECT id, name, description, tags_json, schema_version, created_at, updated_at
		FROM flows WHERE id = ?`, id)
	f := &flow.Flow{}
	var tagsJSON string
	var createdAt, updatedAt string
	err := row.Scan(&f.ID, &f.Name, &f.Description, &tagsJSON, &f.SchemaVersion, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("flow not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(tagsJSON), &f.Tags)
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	steps, err := r.loadSteps(id)
	if err != nil {
		return nil, err
	}
	f.Steps = steps
	return f, nil
}

// Delete removes a flow and its steps (cascade).
func (r *FlowRepo) Delete(id string) error {
	_, err := r.db.Conn.Exec("DELETE FROM flows WHERE id = ?", id)
	return err
}

// List returns all flows without steps.
func (r *FlowRepo) List() ([]*flow.Flow, error) {
	rows, err := r.db.Conn.Query(`
		SELECT id, name, description, tags_json, schema_version, created_at, updated_at
		FROM flows ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flows []*flow.Flow
	for rows.Next() {
		f := &flow.Flow{}
		var tagsJSON string
		var createdAt, updatedAt string
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &tagsJSON, &f.SchemaVersion, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &f.Tags)
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		flows = append(flows, f)
	}
	return flows, rows.Err()
}

// ListSorted returns flows sorted by name.
func (r *FlowRepo) ListSorted() ([]*flow.Flow, error) {
	flows, err := r.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(flows, func(i, j int) bool {
		return strings.ToLower(flows[i].Name) < strings.ToLower(flows[j].Name)
	})
	return flows, nil
}

// Search searches flows by name, description, or tag.
func (r *FlowRepo) Search(query string, tag string) ([]*flow.Flow, error) {
	flows, err := r.List()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var result []*flow.Flow
	for _, f := range flows {
		if tag != "" {
			hasTag := false
			for _, t := range f.Tags {
				if t == tag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		if q == "" {
			result = append(result, f)
			continue
		}
		if strings.Contains(strings.ToLower(f.Name), q) || strings.Contains(strings.ToLower(f.Description), q) {
			result = append(result, f)
			continue
		}
		for _, t := range f.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				result = append(result, f)
				break
			}
		}
	}
	return result, nil
}

func (r *FlowRepo) loadSteps(flowID string) ([]flow.Step, error) {
	rows, err := r.db.Conn.Query(`
		SELECT id, sort_order, name, enabled, type, target_strategy, target_value,
			input_mode, input_text, input_mask_in_logs, wait_before_ms, wait_after_ms, timeout_ms,
			on_error, note
		FROM steps WHERE flow_id = ? ORDER BY sort_order`, flowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []flow.Step
	for rows.Next() {
		var s flow.Step
		var enabled, maskInLogs int
		var stepType, targetStrat, inputMode, onError string
		var sortOrder int
		err := rows.Scan(&s.ID, &sortOrder, &s.Name, &enabled, &stepType, &targetStrat, &s.Target.Value,
			&inputMode, &s.Input.Text, &maskInLogs, &s.WaitBeforeMs, &s.WaitAfterMs, &s.TimeoutMs,
			&onError, &s.Note)
		if err != nil {
			return nil, err
		}
		s.Enabled = enabled != 0
		s.Type = flow.StepType(stepType)
		s.Target.Strategy = flow.TargetStrategy(targetStrat)
		s.Input.Mode = flow.InputMode(inputMode)
		s.Input.MaskInLogs = maskInLogs != 0
		s.OnError = flow.ErrorPolicy(onError)
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
