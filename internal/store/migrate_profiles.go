// Package store provides SQLite persistence and repositories.
//
// Phase 2: full implementation ported from desktop/backend/services/db.go.
// Uses modernc.org/sqlite (pure-Go, no CGo) to keep go.mod minimal.
package store

import (
	"context"

	_ "modernc.org/sqlite"
)

// --- Migrations ------------------------------------------------------------

func (s *Store) migrateProfilesNewColumns(ctx context.Context) {
	cols := []string{
		"ALTER TABLE profiles ADD COLUMN max_age TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN min_age TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN max_depth INTEGER",
		"ALTER TABLE profiles ADD COLUMN delete_excluded INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN dry_run INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN max_transfer TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN max_delete_size TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN suffix TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN suffix_keep_extension INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN check_first INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN order_by TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN retries_sleep TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN tps_limit REAL",
		"ALTER TABLE profiles ADD COLUMN conn_timeout TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN io_timeout TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN size_only INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN update_mode INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN ignore_existing INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN delete_timing TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN resilient INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN max_lock TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN check_access INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE profiles ADD COLUMN conflict_loser TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN conflict_suffix TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE profiles ADD COLUMN direction TEXT NOT NULL DEFAULT ''",
	}
	for _, ddl := range cols {
		// Silently ignore "duplicate column" errors — ALTER TABLE ADD COLUMN is idempotent in spirit.
		_, _ = s.db.ExecContext(ctx, ddl)
	}
}
