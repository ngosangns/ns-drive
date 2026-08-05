// Package store provides SQLite persistence and repositories.
//
// Phase 2: full implementation ported from desktop/backend/services/db.go.
// Uses modernc.org/sqlite (pure-Go, no CGo) to keep go.mod minimal.
package store

import (
	"context"
	"errors"

	_ "modernc.org/sqlite"
)

// --- History repository ----------------------------------------------------

type HistoryRepo struct{ s *Store }

func (s *Store) History() HistoryRepo { return HistoryRepo{s: s} }

func (r HistoryRepo) List(ctx context.Context, limit, offset int) ([]HistoryEntry, error) {
	rows, err := r.s.db.QueryContext(ctx,
		`SELECT id, profile_name, action, status, start_time, end_time, duration,
		        files_transferred, bytes_transferred, errors, error_message
		 FROM history ORDER BY start_time DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHistory(rows)
}

func (r HistoryRepo) ListByProfile(ctx context.Context, profileName string, limit, offset int) ([]HistoryEntry, error) {
	rows, err := r.s.db.QueryContext(ctx,
		`SELECT id, profile_name, action, status, start_time, end_time, duration,
		        files_transferred, bytes_transferred, errors, error_message
		 FROM history WHERE profile_name = ? ORDER BY start_time DESC LIMIT ? OFFSET ?`,
		profileName, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHistory(rows)
}

func (r HistoryRepo) Save(ctx context.Context, e *HistoryEntry) error {
	if e.ID == "" {
		return errors.New("history: id is required")
	}
	_, err := r.s.db.ExecContext(ctx,
		`INSERT INTO history (id, profile_name, action, status, start_time, end_time, duration,
		                     files_transferred, bytes_transferred, errors, error_message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   status=excluded.status, end_time=excluded.end_time, duration=excluded.duration,
		   files_transferred=excluded.files_transferred, bytes_transferred=excluded.bytes_transferred,
		   errors=excluded.errors, error_message=excluded.error_message`,
		e.ID, e.ProfileName, e.Action, e.State, e.StartedAt, e.FinishedAt, e.Duration,
		e.Files, e.Bytes, e.Errors, e.ErrorMessage)
	return err
}

func (r HistoryRepo) Clear(ctx context.Context) error {
	_, err := r.s.db.ExecContext(ctx, "DELETE FROM history")
	return err
}

func (r HistoryRepo) Stats(ctx context.Context) (HistoryStats, error) {
	var stats HistoryStats
	stats.ByProfile = map[string]ProfileStats{}

	row := r.s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(bytes_transferred), 0), COALESCE(SUM(duration), 0), COALESCE(SUM(errors), 0)
		 FROM history`)
	if err := row.Scan(&stats.TotalSyncs, &stats.TotalBytes, &stats.TotalDuration, &stats.TotalErrors); err != nil {
		return stats, err
	}

	rows, err := r.s.db.QueryContext(ctx,
		`SELECT profile_name, COUNT(*), COALESCE(SUM(bytes_transferred), 0),
		        COALESCE(SUM(duration), 0), COALESCE(SUM(errors), 0)
		 FROM history GROUP BY profile_name`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var ps ProfileStats
		if err := rows.Scan(&name, &ps.Syncs, &ps.Bytes, &ps.Duration, &ps.Errors); err != nil {
			return stats, err
		}
		stats.ByProfile[name] = ps
	}
	return stats, rows.Err()
}
