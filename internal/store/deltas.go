// Package store provides SQLite persistence and repositories.
//
// Phase 2: full implementation ported from desktop/backend/services/db.go.
// Uses modernc.org/sqlite (pure-Go, no CGo) to keep go.mod minimal.
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

type DeltaRepo struct{ s *Store }

func (s *Store) Deltas() DeltaRepo { return DeltaRepo{s: s} }

func (r DeltaRepo) GetState(ctx context.Context, remoteKey string) (*DeltaState, error) {
	row := r.s.db.QueryRowContext(ctx,
		`SELECT remote_key, provider, last_full_sync, delta_count, is_watching
		 FROM delta_state WHERE remote_key = ?`, remoteKey)
	var d DeltaState
	var isWatching int
	var lastFull sql.NullString
	if err := row.Scan(&d.RemoteKey, &d.Provider, &lastFull, &d.DeltaCount, &isWatching); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if lastFull.Valid {
		d.LastFullSync = lastFull.String
	}
	d.IsWatching = isWatching != 0
	return &d, nil
}

func (r DeltaRepo) RecordFullSync(ctx context.Context, remoteKey, provider string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.s.db.ExecContext(ctx,
		`INSERT INTO delta_state (remote_key, provider, last_full_sync, delta_count, is_watching)
		 VALUES (?, ?, ?, 0, 0)
		 ON CONFLICT(remote_key) DO UPDATE SET
		   last_full_sync=excluded.last_full_sync, delta_count=0, is_watching=0`,
		remoteKey, provider, now)
	return err
}
