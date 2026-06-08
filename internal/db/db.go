package db

import (
	"database/sql"
	"embed"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string, migrations embed.FS) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("pragma journal_mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}

	if err := runMigrations(db, migrations); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB, fs embed.FS) error {
	var v1 int
	_ = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version=1").Scan(&v1)
	if v1 == 0 {
		data, err := fs.ReadFile("migrations/001_initial.sql")
		if err != nil {
			return fmt.Errorf("read migration 001: %w", err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			return fmt.Errorf("apply migration 001: %w", err)
		}
	}

	var v2 int
	_ = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version=2").Scan(&v2)
	if v2 == 0 {
		data, err := fs.ReadFile("migrations/002_knockout.sql")
		if err != nil {
			return fmt.Errorf("read migration 002: %w", err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			return fmt.Errorf("apply migration 002: %w", err)
		}
	}

	return nil
}
