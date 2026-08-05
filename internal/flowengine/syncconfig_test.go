package flowengine_test

import (
	"encoding/json"
	"testing"

	"github.com/gnasdev/gn-drive/internal/flowengine"
)

// ProfileFromSyncConfig is the shipped mapping used by flow runs.
func TestProfileFromSyncConfig_SnakeAndCamel(t *testing.T) {
	raw := json.RawMessage(`{
		"dry_run": true,
		"parallel": 8,
		"multiThreadStreams": 4,
		"excludedPaths": ["*.tmp", "node_modules"],
		"maxDepth": 3,
		"bandwidth": 10
	}`)
	p := flowengine.ProfileFromSyncConfig(raw)
	if p == nil {
		t.Fatal("nil profile")
	}
	if !p.DryRun {
		t.Error("DryRun want true")
	}
	if p.Parallel != 8 {
		t.Errorf("Parallel = %d, want 8", p.Parallel)
	}
	if p.Bandwidth != 10 {
		t.Errorf("Bandwidth = %d, want 10", p.Bandwidth)
	}
	if p.MultiThreadStreams == nil || *p.MultiThreadStreams != 4 {
		t.Errorf("MultiThreadStreams = %v, want 4", p.MultiThreadStreams)
	}
	if p.MaxDepth == nil || *p.MaxDepth != 3 {
		t.Errorf("MaxDepth = %v, want 3", p.MaxDepth)
	}
	if len(p.ExcludedPaths) != 2 || p.ExcludedPaths[0] != "*.tmp" {
		t.Errorf("ExcludedPaths = %v", p.ExcludedPaths)
	}
}

func TestProfileFromSyncConfig_EmptyDefaults(t *testing.T) {
	p := flowengine.ProfileFromSyncConfig(nil)
	if p.Parallel != 4 {
		t.Errorf("default Parallel = %d, want 4", p.Parallel)
	}
	p2 := flowengine.ProfileFromSyncConfig(json.RawMessage(`not-json`))
	if p2.Parallel != 4 {
		t.Errorf("invalid JSON should still return defaults")
	}
}
