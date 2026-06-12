package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// Environment represents a runtime environment.
type Environment struct {
	ID          string
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EnvironmentVariable represents a variable within an environment.
type EnvironmentVariable struct {
	ID            string
	EnvironmentID string
	Key           string
	Value         string
	IsSecret      bool
	Description   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// EnvRepo handles environment persistence.
type EnvRepo struct {
	db *DB
}

type environmentExport struct {
	Version      int                      `json:"version"`
	ExportedAt   time.Time                `json:"exportedAt"`
	Environments []environmentExportEntry `json:"environments"`
}

type environmentExportEntry struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	IsActive    bool                        `json:"isActive"`
	Variables   []environmentVariableExport `json:"variables"`
}

type environmentVariableExport struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsSecret    bool   `json:"isSecret"`
	Description string `json:"description"`
}

// NewEnvRepo creates an environment repository.
func NewEnvRepo(db *DB) *EnvRepo {
	return &EnvRepo{db: db}
}

// CreateDefaultIfNone creates a default environment if none exist.
func (r *EnvRepo) CreateDefaultIfNone() error {
	var count int
	err := r.db.Conn.QueryRow("SELECT COUNT(*) FROM environments").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().UTC()
	e := &Environment{
		ID:        uuid.New().String(),
		Name:      "默认环境",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return r.Save(e)
}

// Save inserts or updates an environment.
func (r *EnvRepo) Save(e *Environment) error {
	nowTime := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = nowTime
	}
	e.UpdatedAt = nowTime
	now := nowTime.Format(time.RFC3339)
	created := e.CreatedAt.UTC().Format(time.RFC3339)
	_, err := r.db.Conn.Exec(`
		INSERT INTO environments (id, name, description, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			is_active = excluded.is_active,
			updated_at = excluded.updated_at
	`, e.ID, e.Name, e.Description, boolToInt(e.IsActive), created, now)
	return err
}

// Delete removes an environment.
func (r *EnvRepo) Delete(id string) error {
	_, err := r.db.Conn.Exec("DELETE FROM environments WHERE id = ?", id)
	return err
}

// Get loads an environment by ID.
func (r *EnvRepo) Get(id string) (*Environment, error) {
	row := r.db.Conn.QueryRow("SELECT id, name, description, is_active, created_at, updated_at FROM environments WHERE id = ?", id)
	e := &Environment{}
	var isActive int
	var createdAt, updatedAt string
	err := row.Scan(&e.ID, &e.Name, &e.Description, &isActive, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	e.IsActive = isActive != 0
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return e, nil
}

// List returns all environments.
// GetByName loads an environment by name.
func (r *EnvRepo) GetByName(name string) (*Environment, error) {
	row := r.db.Conn.QueryRow("SELECT id, name, description, is_active, created_at, updated_at FROM environments WHERE name = ?", name)
	e := &Environment{}
	var isActive int
	var createdAt, updatedAt string
	err := row.Scan(&e.ID, &e.Name, &e.Description, &isActive, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found: %s", name)
	}
	if err != nil {
		return nil, err
	}
	e.IsActive = isActive != 0
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return e, nil
}

func (r *EnvRepo) List() ([]*Environment, error) {
	rows, err := r.db.Conn.Query("SELECT id, name, description, is_active, created_at, updated_at FROM environments ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var envs []*Environment
	for rows.Next() {
		e := &Environment{}
		var isActive int
		var createdAt, updatedAt string
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &isActive, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		e.IsActive = isActive != 0
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		envs = append(envs, e)
	}
	return envs, rows.Err()
}

// GetActive returns the currently active environment.
func (r *EnvRepo) GetActive() (*Environment, error) {
	row := r.db.Conn.QueryRow("SELECT id, name, description, is_active, created_at, updated_at FROM environments WHERE is_active = 1 LIMIT 1")
	e := &Environment{}
	var isActive int
	var createdAt, updatedAt string
	err := row.Scan(&e.ID, &e.Name, &e.Description, &isActive, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active environment")
	}
	if err != nil {
		return nil, err
	}
	e.IsActive = isActive != 0
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return e, nil
}

// SetActive marks one environment as active and deactivates others.
func (r *EnvRepo) SetActive(id string) error {
	tx, err := r.db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec("UPDATE environments SET is_active = 0")
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE environments SET is_active = 1 WHERE id = ?", id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// --- Environment Variables ---

// SaveVar inserts or updates a variable.
func (r *EnvRepo) SaveVar(v *EnvironmentVariable) error {
	nowTime := time.Now().UTC()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = nowTime
	}
	v.UpdatedAt = nowTime
	now := nowTime.Format(time.RFC3339)
	created := v.CreatedAt.UTC().Format(time.RFC3339)
	_, err := r.db.Conn.Exec(`
		INSERT INTO environment_variables (id, environment_id, key, value, is_secret, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(environment_id, key) DO UPDATE SET
			value = excluded.value,
			is_secret = excluded.is_secret,
			description = excluded.description,
			updated_at = excluded.updated_at
	`, v.ID, v.EnvironmentID, v.Key, v.Value, boolToInt(v.IsSecret), v.Description, created, now)
	return err
}

// DeleteVar removes a variable by ID.
func (r *EnvRepo) DeleteVar(id string) error {
	_, err := r.db.Conn.Exec("DELETE FROM environment_variables WHERE id = ?", id)
	return err
}

// ListVars returns variables for an environment.
func (r *EnvRepo) ListVars(environmentID string) ([]*EnvironmentVariable, error) {
	rows, err := r.db.Conn.Query(`
		SELECT id, environment_id, key, value, is_secret, description, created_at, updated_at
		FROM environment_variables WHERE environment_id = ? ORDER BY key`, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []*EnvironmentVariable
	for rows.Next() {
		v := &EnvironmentVariable{}
		var isSecret int
		var createdAt, updatedAt string
		if err := rows.Scan(&v.ID, &v.EnvironmentID, &v.Key, &v.Value, &isSecret, &v.Description, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		v.IsSecret = isSecret != 0
		v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		v.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		vars = append(vars, v)
	}
	return vars, rows.Err()
}

// GetVar returns a single variable by environment and key.
func (r *EnvRepo) GetVar(environmentID, key string) (*EnvironmentVariable, error) {
	row := r.db.Conn.QueryRow(`
		SELECT id, environment_id, key, value, is_secret, description, created_at, updated_at
		FROM environment_variables WHERE environment_id = ? AND key = ?`, environmentID, key)
	v := &EnvironmentVariable{}
	var isSecret int
	var createdAt, updatedAt string
	err := row.Scan(&v.ID, &v.EnvironmentID, &v.Key, &v.Value, &isSecret, &v.Description, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("variable not found: %s", key)
	}
	if err != nil {
		return nil, err
	}
	v.IsSecret = isSecret != 0
	v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	v.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return v, nil
}

// EnvProvider implements template.EnvProvider.
func (r *EnvRepo) EnvProvider(environmentID string) *envProvider {
	return &envProvider{repo: r, envID: environmentID}
}

// Export writes all environments and variables to a JSON file.
func (r *EnvRepo) Export(path string) error {
	envs, err := r.List()
	if err != nil {
		return err
	}
	out := environmentExport{
		Version:    1,
		ExportedAt: time.Now().UTC(),
	}
	for _, env := range envs {
		vars, err := r.ListVars(env.ID)
		if err != nil {
			return err
		}
		entry := environmentExportEntry{
			Name:        env.Name,
			Description: env.Description,
			IsActive:    env.IsActive,
		}
		for _, v := range vars {
			entry.Variables = append(entry.Variables, environmentVariableExport{
				Key:         v.Key,
				Value:       v.Value,
				IsSecret:    v.IsSecret,
				Description: v.Description,
			})
		}
		out.Environments = append(out.Environments, entry)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Import reads environments and variables from a JSON file. Existing
// environments are never overwritten; if a name already exists, the imported
// environment is renamed with a suffix such as "默认环境 导入".
func (r *EnvRepo) Import(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var in environmentExport
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}

	existingNames := make(map[string]bool)
	envs, _ := r.List()
	for _, e := range envs {
		existingNames[e.Name] = true
	}

	for _, entry := range in.Environments {
		if entry.Name == "" {
			continue
		}
		name := entry.Name
		if existingNames[name] {
			name = r.uniqueEnvName(name, existingNames)
		}
		existingNames[name] = true

		env := &Environment{ID: uuid.New().String(), Name: name}
		env.Description = entry.Description
		env.IsActive = entry.IsActive
		if err := r.Save(env); err != nil {
			return err
		}
		if entry.IsActive {
			if err := r.SetActive(env.ID); err != nil {
				return err
			}
		}
		for _, v := range entry.Variables {
			if v.Key == "" {
				continue
			}
			existing, err := r.GetVar(env.ID, v.Key)
			if err != nil {
				existing = &EnvironmentVariable{ID: uuid.New().String(), EnvironmentID: env.ID, Key: v.Key}
			}
			existing.Value = v.Value
			existing.IsSecret = v.IsSecret
			existing.Description = v.Description
			if err := r.SaveVar(existing); err != nil {
				return err
			}
		}
	}
	return r.CreateDefaultIfNone()
}

// uniqueEnvName returns a name that does not collide with any existing
// environment. It appends " 导入", " 导入 2", etc.
func (r *EnvRepo) uniqueEnvName(base string, existing map[string]bool) string {
	candidate := base + " 导入"
	if !existing[candidate] {
		return candidate
	}
	for i := 2; ; i++ {
		candidate = fmt.Sprintf("%s 导入 %d", base, i)
		if !existing[candidate] {
			return candidate
		}
	}
}

// envProvider adapts EnvRepo to template engine.
type envProvider struct {
	repo  *EnvRepo
	envID string
}

// GetEnvValue returns the value, found flag, and secret flag for a key.
func (p *envProvider) GetEnvValue(key string) (string, bool, bool) {
	v, err := p.repo.GetVar(p.envID, key)
	if err != nil {
		return "", false, false
	}
	return v.Value, true, v.IsSecret
}
