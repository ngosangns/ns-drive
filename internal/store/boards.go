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

// --- Board / Flow / Delta repositories (Phase 3 wires full CRUD) ---------

type BoardRepo struct{ s *Store }

func (s *Store) Boards() BoardRepo { return BoardRepo{s: s} }

func (r BoardRepo) List(ctx context.Context) ([]Board, error) {
	rows, err := r.s.db.QueryContext(ctx,
		`SELECT id, name, created_at, updated_at, schedule_enabled, cron_expr
		 FROM boards ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []Board
	for rows.Next() {
		var b Board
		var schedEnabled int
		var cron sql.NullString
		if err := rows.Scan(&b.ID, &b.Name, &b.CreatedAt, &b.UpdatedAt,
			&schedEnabled, &cron); err != nil {
			return nil, err
		}
		// Nodes/edges loaded via LoadGraph.
		boards = append(boards, b)
	}
	return boards, rows.Err()
}

func (r BoardRepo) Get(ctx context.Context, id string) (*Board, error) {
	row := r.s.db.QueryRowContext(ctx,
		`SELECT id, name, created_at, updated_at, schedule_enabled, cron_expr
		 FROM boards WHERE id = ?`, id)
	var b Board
	var schedEnabled int
	var cron sql.NullString
	if err := row.Scan(&b.ID, &b.Name, &b.CreatedAt, &b.UpdatedAt,
		&schedEnabled, &cron); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

// LoadGraph returns the board with its nodes and edges populated. Used for
// DAG execution and the web UI canvas.
func (r BoardRepo) LoadGraph(ctx context.Context, id string) (*Board, error) {
	b, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Nodes = []BoardNode{}
	b.Edges = []BoardEdge{}

	nrows, err := r.s.db.QueryContext(ctx,
		`SELECT id, remote_name, path, label, x, y
		 FROM board_nodes WHERE board_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer nrows.Close()
	for nrows.Next() {
		var n BoardNode
		if err := nrows.Scan(&n.ID, &n.RemoteName, &n.Path, &n.Label, &n.X, &n.Y); err != nil {
			return nil, err
		}
		b.Nodes = append(b.Nodes, n)
	}
	if err := nrows.Err(); err != nil {
		return nil, err
	}

	erows, err := r.s.db.QueryContext(ctx,
		`SELECT id, source_id, target_id, action, sync_config
		 FROM board_edges WHERE board_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var e BoardEdge
		var syncCfg string
		if err := erows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.Action, &syncCfg); err != nil {
			return nil, err
		}
		e.SyncConfig = json.RawMessage(syncCfg)
		b.Edges = append(b.Edges, e)
	}
	return b, erows.Err()
}

func (r BoardRepo) Save(ctx context.Context, b *Board) error {
	_, err := r.s.db.ExecContext(ctx,
		`INSERT INTO boards (id, name, schedule_enabled, cron_expr, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name,
		   schedule_enabled=excluded.schedule_enabled, cron_expr=excluded.cron_expr,
		   updated_at=datetime('now')`,
		b.ID, b.Name, 0, "")
	return err
}

// SaveGraph persists the board along with its nodes and edges in a single
// transaction. It deletes any prior nodes/edges for the board.
func (r BoardRepo) SaveGraph(ctx context.Context, b *Board) error {
	tx, err := r.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO boards (id, name, schedule_enabled, cron_expr, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'))
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name,
		   schedule_enabled=excluded.schedule_enabled, cron_expr=excluded.cron_expr,
		   updated_at=datetime('now')`,
		b.ID, b.Name, 0, ""); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM board_nodes WHERE board_id = ?", b.ID); err != nil {
		return err
	}
	for _, n := range b.Nodes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO board_nodes (id, board_id, remote_name, path, label, x, y)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			n.ID, b.ID, n.RemoteName, n.Path, n.Label, n.X, n.Y); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM board_edges WHERE board_id = ?", b.ID); err != nil {
		return err
	}
	for _, e := range b.Edges {
		cfg := string(e.SyncConfig)
		if cfg == "" {
			cfg = "{}"
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO board_edges (id, board_id, source_id, target_id, action, sync_config)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			e.ID, b.ID, e.SourceID, e.TargetID, e.Action, cfg); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r BoardRepo) Delete(ctx context.Context, id string) error {
	res, err := r.s.db.ExecContext(ctx, "DELETE FROM boards WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
