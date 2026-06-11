package db

// SchemaSQL contains the initial schema creation statements.
// Migrations are tracked in schema_migrations table.
const SchemaSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS app_state (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS flows (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  schema_version INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS steps (
  id TEXT PRIMARY KEY,
  flow_id TEXT NOT NULL,
  sort_order INTEGER NOT NULL,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  type TEXT NOT NULL,
  target_strategy TEXT NOT NULL DEFAULT 'xpath',
  target_value TEXT NOT NULL DEFAULT '',
  input_mode TEXT NOT NULL DEFAULT 'template',
  input_text TEXT NOT NULL DEFAULT '',
  input_mask_in_logs INTEGER NOT NULL DEFAULT 0,
  wait_before_ms INTEGER NOT NULL DEFAULT 0,
  wait_after_ms INTEGER NOT NULL DEFAULT 500,
  timeout_ms INTEGER NOT NULL DEFAULT 10000,
  on_error TEXT NOT NULL DEFAULT 'stop',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY(flow_id) REFERENCES flows(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS environments (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  is_active INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS environment_variables (
  id TEXT PRIMARY KEY,
  environment_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL DEFAULT '',
  is_secret INTEGER NOT NULL DEFAULT 0,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(environment_id, key),
  FOREIGN KEY(environment_id) REFERENCES environments(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_results (
  id TEXT PRIMARY KEY,
  flow_id TEXT NOT NULL,
  environment_id TEXT,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  total_steps INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  failed_count INTEGER NOT NULL DEFAULT 0,
  skipped_count INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  snapshot_dir TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(flow_id) REFERENCES flows(id) ON DELETE CASCADE,
  FOREIGN KEY(environment_id) REFERENCES environments(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS run_step_results (
  id TEXT PRIMARY KEY,
  run_result_id TEXT NOT NULL,
  step_id TEXT,
  step_order INTEGER NOT NULL,
  step_name TEXT NOT NULL,
  step_type TEXT NOT NULL,
  status TEXT NOT NULL,
  error TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0,
  screenshot_path TEXT NOT NULL DEFAULT '',
  html_snapshot_path TEXT NOT NULL DEFAULT '',
  generated_input_masked TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  FOREIGN KEY(run_result_id) REFERENCES run_results(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_steps_flow_order ON steps(flow_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_env_vars_environment ON environment_variables(environment_id);
CREATE INDEX IF NOT EXISTS idx_run_results_flow_started ON run_results(flow_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_run_step_results_run_order ON run_step_results(run_result_id, step_order);
`
