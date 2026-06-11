package db

import (
	"database/sql"
	"fmt"
	"time"
)

const currentSchemaVersion = 1

// Migrate applies pending schema migrations.
func (db *DB) Migrate() error {
	if _, err := db.Conn.Exec(SchemaSQL); err != nil {
		return fmt.Errorf("apply base schema: %w", err)
	}

	version, err := db.currentVersion()
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if version >= currentSchemaVersion {
		return nil
	}

	for v := version + 1; v <= currentSchemaVersion; v++ {
		if err := applyMigration(db.Conn, v); err != nil {
			return fmt.Errorf("apply migration %d: %w", v, err)
		}
	}
	return nil
}

func (db *DB) currentVersion() (int, error) {
	var version int
	err := db.Conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return version, nil
}

func applyMigration(conn *sql.DB, version int) error {
	switch version {
	case 1:
		// Initial schema already applied by SchemaSQL.
		// Future migrations add ALTER TABLE or new tables here.
	default:
		return fmt.Errorf("unknown migration version %d", version)
	}

	_, err := conn.Exec(
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
		version, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}
