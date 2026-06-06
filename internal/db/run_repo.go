package db

import (
	"database/sql"
	"time"

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
	_, err = tx.Exec(`
		INSERT INTO run_results (id, flow_id, environment_id, status, started_at, finished_at,
			total_steps, success_count, failed_count, skipped_count, duration_ms, snapshot_dir)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, res.ID, res.FlowID, environmentID, string(res.Status),
		res.StartedAt.UTC().Format(time.RFC3339), res.FinishedAt.UTC().Format(time.RFC3339),
		res.TotalSteps, res.SuccessCount, res.FailedCount, res.SkippedCount, durationMs, "")
	if err != nil {
		return err
	}

	for _, s := range res.Steps {
		_, err = tx.Exec(`
			INSERT INTO run_step_results (id, run_result_id, step_id, step_order, step_name, step_type,
				status, error, duration_ms, screenshot_path, html_snapshot_path, generated_input_masked, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.StepID, res.ID, s.StepID, 0, s.StepName, s.Type, string(s.Status), s.Error,
			s.DurationMs, s.Screenshot, s.HTMLSnapshot, s.GeneratedInput, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListByFlow returns run results for a flow, newest first.
func (r *RunRepo) ListByFlow(flowID string, limit int) ([]*runner.RunResult, error) {
	query := `
		SELECT id, flow_id, environment_id, status, started_at, finished_at,
			total_steps, success_count, failed_count, skipped_count, duration_ms
		FROM run_results WHERE flow_id = ? ORDER BY started_at DESC`
	args := []interface{}{flowID}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
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
		res.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		res.FinishedAt, _ = time.Parse(time.RFC3339, finishedAt)
		results = append(results, res)
	}
	return results, rows.Err()
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
		res.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		res.FinishedAt, _ = time.Parse(time.RFC3339, finishedAt)
		results = append(results, res)
	}
	return results, rows.Err()
}

// Cleanup deletes run results older than the given number of days.
func (r *RunRepo) Cleanup(days int) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	_, err := r.db.Conn.Exec("DELETE FROM run_results WHERE started_at < ?", cutoff)
	return err
}
