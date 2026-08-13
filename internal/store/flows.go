// Package store provides SQLite persistence and repositories.
//
// Phase 2: full implementation ported from desktop/backend/services/db.go.
// Uses modernc.org/sqlite (pure-Go, no CGo) to keep go.mod minimal.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	_ "modernc.org/sqlite"
)

type FlowRepo struct{ s *Store }

func (s *Store) Flows() FlowRepo { return FlowRepo{s: s} }

func (r FlowRepo) List(ctx context.Context) ([]Flow, error) {
	rows, err := r.s.db.QueryContext(ctx,
		`SELECT id, name, is_collapsed, schedule_enabled, cron_expr, sort_order, created_at, updated_at, canvas_json
		 FROM flows ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flows []Flow
	for rows.Next() {
		f, err := scanFlow(rows)
		if err != nil {
			return nil, err
		}
		flows = append(flows, *f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Load operations after closing flow rows — SQLite is single-conn.
	for i := range flows {
		ops, err := r.listOperations(ctx, flows[i].ID)
		if err != nil {
			return nil, err
		}
		flows[i].Operations = ops
	}
	if flows == nil {
		flows = []Flow{}
	}
	return flows, nil
}

func (r FlowRepo) Get(ctx context.Context, id string) (*Flow, error) {
	row := r.s.db.QueryRowContext(ctx,
		`SELECT id, name, is_collapsed, schedule_enabled, cron_expr, sort_order, created_at, updated_at, canvas_json
		 FROM flows WHERE id = ?`, id)
	f, err := scanFlow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	ops, err := r.listOperations(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	f.Operations = ops
	return f, nil
}

func scanFlow(r rowScanner) (*Flow, error) {
	var f Flow
	var collapsed, schedEnabled, sortOrder int
	var cron, createdAt, updatedAt, canvas sql.NullString
	if err := r.Scan(&f.ID, &f.Name, &collapsed, &schedEnabled, &cron, &sortOrder, &createdAt, &updatedAt, &canvas); err != nil {
		return nil, err
	}
	f.IsCollapsed = collapsed != 0
	f.ScheduleEnabled = schedEnabled != 0
	f.Enabled = f.ScheduleEnabled
	f.SortOrder = sortOrder
	if cron.Valid {
		f.ScheduleCron = cron.String
		f.CronExpr = cron.String
	}
	if createdAt.Valid {
		f.CreatedAt = createdAt.String
	}
	if updatedAt.Valid {
		f.UpdatedAt = updatedAt.String
	}
	if canvas.Valid && canvas.String != "" {
		f.CanvasJSON = json.RawMessage(canvas.String)
	}
	if f.Operations == nil {
		f.Operations = []Operation{}
	}
	return &f, nil
}

func (r FlowRepo) listOperations(ctx context.Context, flowID string) ([]Operation, error) {
	rows, err := r.s.db.QueryContext(ctx,
		`SELECT id, flow_id, source_remote, source_path, target_remote, target_path,
		        action, sync_config, is_expanded, sort_order
		 FROM operations WHERE flow_id = ? ORDER BY sort_order, id`, flowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ops []Operation
	for rows.Next() {
		var op Operation
		var expanded int
		var cfg sql.NullString
		if err := rows.Scan(
			&op.ID, &op.FlowID, &op.SourceRemote, &op.SourcePath,
			&op.TargetRemote, &op.TargetPath, &op.Action, &cfg,
			&expanded, &op.SortOrder,
		); err != nil {
			return nil, err
		}
		op.IsExpanded = expanded != 0
		if cfg.Valid && cfg.String != "" {
			op.SyncConfig = json.RawMessage(cfg.String)
		}
		// Align action column with sync_config.action (Wails load mapping).
		op.NormalizeAction()
		ops = append(ops, op)
	}
	if ops == nil {
		ops = []Operation{}
	}
	return ops, rows.Err()
}

// Save upserts a flow and replaces its operations (Wails SaveFlows semantics per flow).
func (r FlowRepo) Save(ctx context.Context, f *Flow) error {
	if f == nil || f.ID == "" {
		return errors.New("flow: id is required")
	}
	// Normalize schedule flags from either field name clients may send.
	schedEnabled := f.ScheduleEnabled || f.Enabled
	cron := f.ScheduleCron
	if cron == "" {
		cron = f.CronExpr
	}
	f.ScheduleEnabled = schedEnabled
	f.Enabled = schedEnabled
	f.ScheduleCron = cron
	f.CronExpr = cron

	tx, err := r.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO flows (id, name, is_collapsed, schedule_enabled, cron_expr, sort_order, created_at, updated_at, canvas_json)
		 VALUES (?, ?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), datetime('now')), datetime('now'), ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name, is_collapsed=excluded.is_collapsed,
		   schedule_enabled=excluded.schedule_enabled, cron_expr=excluded.cron_expr,
		   sort_order=excluded.sort_order, updated_at=datetime('now'),
		   canvas_json=excluded.canvas_json`,
		// cron_expr is NOT NULL DEFAULT ''; never pass SQL NULL via nullableString.
		f.ID, f.Name, boolToInt(f.IsCollapsed), boolToInt(schedEnabled),
		cron, f.SortOrder, f.CreatedAt, canvasJSONOrEmpty(f.CanvasJSON))
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM operations WHERE flow_id = ?`, f.ID); err != nil {
		return err
	}

	for i := range f.Operations {
		op := &f.Operations[i]
		if op.ID == "" {
			return errors.New("flow: operation id is required")
		}
		op.FlowID = f.ID
		if op.SortOrder == 0 {
			op.SortOrder = i
		}
		// Wails: bo.action = op.syncConfig.action; keep column + JSON in sync.
		op.NormalizeAction()
		cfg := "{}"
		if len(op.SyncConfig) > 0 {
			cfg = string(op.SyncConfig)
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO operations
			 (id, flow_id, source_remote, source_path, target_remote, target_path, action, sync_config, is_expanded, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			op.ID, f.ID, op.SourceRemote, nullIfEmpty(op.SourcePath, "/"),
			op.TargetRemote, nullIfEmpty(op.TargetPath, "/"),
			op.Action, cfg, boolToInt(op.IsExpanded), op.SortOrder,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func canvasJSONOrEmpty(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func nullIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (r FlowRepo) Delete(ctx context.Context, id string) error {
	// operations cascade via FK
	res, err := r.s.db.ExecContext(ctx, "DELETE FROM flows WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
