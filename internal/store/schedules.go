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

// --- Schedule repository ---------------------------------------------------

type ScheduleRepo struct{ s *Store }

func (s *Store) Schedules() ScheduleRepo { return ScheduleRepo{s: s} }

func (r ScheduleRepo) List(ctx context.Context) ([]Schedule, error) {
	rows, err := r.s.db.QueryContext(ctx,
		`SELECT id, profile_name, action, cron_expr, enabled, last_run, next_run, last_result, created_at
		 FROM schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Schedule
	for rows.Next() {
		var s Schedule
		var enabled int
		var lastRun, nextRun, createdAt sql.NullString
		if err := rows.Scan(&s.ID, &s.ProfileName, &s.Action, &s.Cron, &enabled,
			&lastRun, &nextRun, &s.LastResult, &createdAt); err != nil {
			return nil, err
		}
		s.Enabled = enabled != 0
		if lastRun.Valid {
			s.LastRun = lastRun.String
		}
		if nextRun.Valid {
			s.NextRun = nextRun.String
		}
		if createdAt.Valid {
			s.CreatedAt = createdAt.String
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r ScheduleRepo) Get(ctx context.Context, id string) (*Schedule, error) {
	row := r.s.db.QueryRowContext(ctx,
		`SELECT id, profile_name, action, cron_expr, enabled, last_run, next_run, last_result, created_at
		 FROM schedules WHERE id = ?`, id)
	var s Schedule
	var enabled int
	var lastRun, nextRun, lastResult, createdAt sql.NullString
	if err := row.Scan(&s.ID, &s.ProfileName, &s.Action, &s.Cron, &enabled,
		&lastRun, &nextRun, &lastResult, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.Enabled = enabled != 0
	if lastRun.Valid {
		s.LastRun = lastRun.String
	}
	if nextRun.Valid {
		s.NextRun = nextRun.String
	}
	if lastResult.Valid {
		s.LastResult = lastResult.String
	}
	if createdAt.Valid {
		s.CreatedAt = createdAt.String
	}
	return &s, nil
}

func (r ScheduleRepo) Save(ctx context.Context, sch *Schedule) error {
	if sch.ID == "" {
		return errors.New("schedule: id is required")
	}
	_, err := r.s.db.ExecContext(ctx,
		`INSERT INTO schedules (id, profile_name, action, cron_expr, enabled, last_run, next_run, last_result)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   profile_name=excluded.profile_name, action=excluded.action,
		   cron_expr=excluded.cron_expr, enabled=excluded.enabled,
		   last_run=excluded.last_run, next_run=excluded.next_run,
		   last_result=excluded.last_result`,
		sch.ID, sch.ProfileName, sch.Action, sch.Cron, boolToInt(sch.Enabled),
		nullableString(sch.LastRun), nullableString(sch.NextRun), sch.LastResult)
	return err
}

func (r ScheduleRepo) Delete(ctx context.Context, id string) error {
	res, err := r.s.db.ExecContext(ctx, "DELETE FROM schedules WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
