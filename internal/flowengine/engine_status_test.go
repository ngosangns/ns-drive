package flowengine

import "testing"

func TestStatuses_Empty(t *testing.T) {
	e := New(Options{})
	if got := e.Statuses(); len(got) != 0 {
		t.Fatalf("Statuses() = %v, want empty", got)
	}
	if e.Status("missing") != "idle" {
		t.Fatalf("Status(missing) = %q, want idle", e.Status("missing"))
	}
}
