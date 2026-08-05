// Package store provides SQLite persistence and repositories.
//
// Phase 2: full implementation ported from desktop/backend/services/db.go.
// Uses modernc.org/sqlite (pure-Go, no CGo) to keep go.mod minimal.
package store

import (
	"context"
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
)

// --- Settings repository ----------------------------------------------------

type SettingsRepo struct{ s *Store }

func (s *Store) Settings() SettingsRepo { return SettingsRepo{s: s} }

func (r SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.s.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (r SettingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

func (r SettingsRepo) GetBool(ctx context.Context, key string, def bool) bool {
	v, err := r.Get(ctx, key)
	if err != nil {
		return def
	}
	return v == "true" || v == "1"
}
