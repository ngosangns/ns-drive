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
	"strings"

	_ "modernc.org/sqlite"
)

// --- Profile repository ----------------------------------------------------

type ProfileRepo struct{ s *Store }

func (s *Store) Profiles() ProfileRepo { return ProfileRepo{s: s} }

func (r ProfileRepo) List(ctx context.Context) ([]Profile, error) {
	rows, err := r.s.db.QueryContext(ctx, "SELECT "+profileSelectColumns+" FROM profiles ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfiles(rows)
}

func (r ProfileRepo) Get(ctx context.Context, name string) (*Profile, error) {
	row := r.s.db.QueryRowContext(ctx, "SELECT "+profileSelectColumns+" FROM profiles WHERE name = ?", name)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r ProfileRepo) Save(ctx context.Context, p *Profile) error {
	if p.Name == "" {
		return errors.New("profile: name is required")
	}
	// Profiles only accept push | bi | bi-resync (not pull / copy / …).
	// Empty is normalized to push so older clients without a direction still save.
	dir := strings.TrimSpace(p.Direction)
	if dir == "" {
		dir = ProfileDirectionPush
	}
	if !IsValidProfileDirection(dir) {
		return fmt.Errorf("profile: invalid direction %q (allowed: push, bi, bi-resync)", p.Direction)
	}
	p.Direction = dir
	_, err := r.s.db.ExecContext(ctx, profileUpsertSQL,
		p.Name, p.From, p.To, p.Direction,
		marshalStringSlice(p.IncludedPaths), marshalStringSlice(p.ExcludedPaths),
		p.Bandwidth, p.Parallel, p.BackupPath, p.CachePath,
		p.MinSize, p.MaxSize, p.FilterFromFile, p.ExcludeIfPresent,
		boolToInt(p.UseRegex), intPtrToNullable(p.MaxDelete), boolToInt(p.Immutable),
		p.ConflictResolution, intPtrToNullable(p.MultiThreadStreams),
		p.BufferSize, boolToInt(p.FastList),
		intPtrToNullable(p.Retries), intPtrToNullable(p.LowLevelRetries), p.MaxDuration,
		p.MaxAge, p.MinAge, intPtrToNullable(p.MaxDepth), boolToInt(p.DeleteExcluded),
		boolToInt(p.DryRun), p.MaxTransfer, p.MaxDeleteSize, p.Suffix, boolToInt(p.SuffixKeepExtension),
		boolToInt(p.CheckFirst), p.OrderBy, p.RetriesSleep, floatPtrToNullable(p.TpsLimit),
		p.ConnTimeout, p.IoTimeout, boolToInt(p.SizeOnly), boolToInt(p.UpdateMode),
		boolToInt(p.IgnoreExisting), p.DeleteTiming, boolToInt(p.Resilient),
		p.MaxLock, boolToInt(p.CheckAccess), p.ConflictLoser, p.ConflictSuffix,
	)
	return err
}

func (r ProfileRepo) Delete(ctx context.Context, name string) error {
	res, err := r.s.db.ExecContext(ctx, "DELETE FROM profiles WHERE name = ?", name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
