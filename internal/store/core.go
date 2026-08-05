// Package store provides SQLite persistence and repositories.
//
// Phase 2: full implementation ported from desktop/backend/services/db.go.
// Uses modernc.org/sqlite (pure-Go, no CGo) to keep go.mod minimal.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a record is not found.
var ErrNotFound = errors.New("store: record not found")

// Store manages the SQLite database connection and provides repositories.
type Store struct {
	db     *sql.DB
	logger *slog.Logger
	mu     sync.Mutex // serialize schema changes
}

// sqlOpenFn is overridable for tests; defaults to sql.Open.
var sqlOpenFn = func(driver, path string) (*sql.DB, error) {
	return sql.Open(driver, path)
}

// execContextFn is overridable for tests; defaults to (*sql.DB).ExecContext.
var execContextFn = func(db *sql.DB, ctx context.Context, q string) (sql.Result, error) {
	return db.ExecContext(ctx, q)
}

// New opens the SQLite database at the given path, applies migrations,
// and returns a Store with all repositories ready.
func New(ctx context.Context, dbPath string, logger *slog.Logger) (*Store, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sqlOpenFn("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := execContextFn(db, ctx, "PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := execContextFn(db, ctx, "PRAGMA foreign_keys=ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite single-writer

	s := &Store{db: db, logger: logger}
	if err := migrateFn(s, ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	logger.Info("database opened", "path", dbPath)
	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB returns the underlying *sql.DB. Used for transactions in repositories.
func (s *Store) DB() *sql.DB { return s.db }

// migrateFn is overridable for tests; defaults to (*Store).migrate.
var migrateFn = func(s *Store, ctx context.Context) error { return s.migrate(ctx) }

// migrate creates all tables and applies schema migrations.
func (s *Store) migrate(ctx context.Context) error {
	if err := s.createAllTables(ctx); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	s.migrateProfilesNewColumns(ctx)
	if err := s.applyMigrations(ctx); err != nil {
		return fmt.Errorf("apply versioned migrations: %w", err)
	}
	return nil
}

func (s *Store) createAllTables(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// schemaVersion is the current schema version stamped into the database's
// PRAGMA user_version. Bump it whenever you append a migration below.
const schemaVersion = 1

// migration is a single forward schema change, applied when the database's
// user_version is below its version.
type migration struct {
	version int
	sql     string
}

// migrations are applied in ascending version order. Version 1 is the baseline
// schema already materialised by createAllTables, so it carries no extra SQL.
// Add future changes as {version: 2, sql: "ALTER TABLE ..."} and bump
// schemaVersion to match.
var migrations = []migration{
	{version: 1, sql: ""},
}

// applyMigrations runs any forward migrations whose version exceeds the
// database's current user_version, stamping user_version as it goes. It is
// idempotent: a fully-migrated database is a no-op.
func (s *Store) applyMigrations(ctx context.Context) error {
	var current int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if m.sql != "" {
			if _, err := s.db.ExecContext(ctx, m.sql); err != nil {
				return fmt.Errorf("migration v%d: %w", m.version, err)
			}
		}
		// PRAGMA user_version does not accept bound parameters; m.version is an
		// internal constant (never user input), so the formatted value is safe.
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
			return fmt.Errorf("stamp user_version=%d: %w", m.version, err)
		}
		current = m.version
		s.logger.Info("store: applied migration", "version", m.version)
	}
	return nil
}
