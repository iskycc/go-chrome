package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"go-chrome/internal/runner"
)

// RunRepo handles run result persistence.
type RunRepo struct {
	db *DB
}

// NewRunRepo creates a run result repository.
func NewRunRepo(db *DB) *RunRepo {
	return &RunRepo{db: db}
}

// Save inserts a run result and its step results.
func (r *RunRepo) Save(res *runner.RunResult, environmentID string) error {
	tx, err := r.db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	durationMs := int(res.FinishedAt.Sub(res.StartedAt).Milliseconds())
	var envID interface{}
	if environmentID != "" {
		envID = environmentID
	}
	_, err = tx.Exec(`
		INSERT INTO run_results (id, flow_id, environment_id, status, started_at, finished_at,
			total_steps, success_count, failed_count, skipped_count, duration_ms, snapshot_dir)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, res.ID, res.FlowID, envID, string(res.Status),
		res.StartedAt.UTC().Format(time.RFC3339), res.FinishedAt.UTC().Format(time.RFC3339),
		res.TotalSteps, res.SuccessCount, res.FailedCount, res.SkippedCount, durationMs, "")
	if err != nil {
		return err
	}

	for i, s := range res.Steps {
		_, err = tx.Exec(`
			INSERT INTO run_step_results (id, run_result_id, step_id, step_order, step_name, step_type,
				status, error, duration_ms, screenshot_path, html_snapshot_path, generated_input_masked, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), res.ID, s.StepID, i, s.StepName, s.Type, string(s.Status), s.Error,
			s.DurationMs, s.Screenshot, s.HTMLSnapshot, s.GeneratedInputMasked, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListByFlow returns run results for a flow, newest first.
func (r *RunRepo) ListByFlow(flowID string, limit int) ([]*runner.RunResult, error) {
	return r.ListFiltered(flowID, "", "", limit)
}

// ListFiltered returns run results matching optional flow, environment, and status filters.
func (r *RunRepo) ListFiltered(flowID, environmentID string, status runner.Status, limit int) ([]*runner.RunResult, error) {
	query := `
		SELECT id, flow_id, environment_id, status, started_at, finished_at,
			total_steps, success_count, failed_count, skipped_count, duration_ms
		FROM run_results WHERE 1=1`
	args := []interface{}{}
	if flowID != "" {
		query += " AND flow_id = ?"
		args = append(args, flowID)
	}
	if environmentID != "" {
		query += " AND environment_id = ?"
		args = append(args, environmentID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY started_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	return r.list(query, args...)
}

// ListAll returns all run results, newest first.
func (r *RunRepo) ListAll(limit int) ([]*runner.RunResult, error) {
	query := `
		SELECT id, flow_id, environment_id, status, started_at, finished_at,
			total_steps, success_count, failed_count, skipped_count, duration_ms
		FROM run_results ORDER BY started_at DESC`
	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	return r.list(query, args...)
}

func (r *RunRepo) list(query string, args ...interface{}) ([]*runner.RunResult, error) {
	rows, err := r.db.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*runner.RunResult
	for rows.Next() {
		res := &runner.RunResult{}
		var envID sql.NullString
		var startedAt, finishedAt string
		var durationMs int
		err := rows.Scan(&res.ID, &res.FlowID, &envID, &res.Status, &startedAt, &finishedAt,
			&res.TotalSteps, &res.SuccessCount, &res.FailedCount, &res.SkippedCount, &durationMs)
		if err != nil {
			return nil, err
		}
		if envID.Valid {
			res.EnvironmentID = envID.String
		}
		res.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		res.FinishedAt, _ = time.Parse(time.RFC3339, finishedAt)
		res.Steps, err = r.listSteps(res.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, rows.Err()
}

func (r *RunRepo) listSteps(runResultID string) ([]runner.StepResult, error) {
	rows, err := r.db.Conn.Query(`
		SELECT step_id, step_name, step_type, status, error, duration_ms,
			screenshot_path, html_snapshot_path, generated_input_masked
		FROM run_step_results WHERE run_result_id = ? ORDER BY step_order`, runResultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []runner.StepResult
	for rows.Next() {
		var s runner.StepResult
		if err := rows.Scan(&s.StepID, &s.StepName, &s.Type, &s.Status, &s.Error, &s.DurationMs,
			&s.Screenshot, &s.HTMLSnapshot, &s.GeneratedInputMasked); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

// Cleanup deletes run results older than the given number of days.
func (r *RunRepo) Cleanup(days int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	_, err := r.db.Conn.Exec("DELETE FROM run_results WHERE started_at < ?", cutoff)
	return err
}
