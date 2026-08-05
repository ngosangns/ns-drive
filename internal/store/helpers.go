// Package store provides SQLite persistence and repositories.
//
// Phase 2: full implementation ported from desktop/backend/services/db.go.
// Uses modernc.org/sqlite (pure-Go, no CGo) to keep go.mod minimal.
package store

import (
	"encoding/json"

	_ "modernc.org/sqlite"
)

// --- Helpers ---------------------------------------------------------------

func marshalStringSlice(s []string) string {
	if s == nil {
		return "[]"
	}
	b, _ := json.Marshal(s)
	return string(b)
}

func unmarshalStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intPtrToNullable(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func floatPtrToNullable(p *float64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
