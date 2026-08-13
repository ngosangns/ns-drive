package flowengine

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gnasdev/gn-drive/internal/eventbus"
	"github.com/gnasdev/gn-drive/internal/rclone"
	"github.com/gnasdev/gn-drive/internal/store"
	"github.com/gnasdev/gn-drive/internal/syncengine"
)

func writeSleepRclone(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "rclone")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 20\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func writeOkRclone(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "rclone")
	script := "#!/bin/sh\ncase \" $* \" in\n  *lsjson*) echo '[]';;\n  *) echo Transferred: 0 / 0 Bytes;;\nesac\nexit 0\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func testStack(t *testing.T, rcloneBin string) (*Engine, *store.Store, *syncengine.Engine) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, err := store.New(context.Background(), filepath.Join(t.TempDir(), "db.db"), log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rc, err := rclone.New(rclone.Options{BinaryPath: rcloneBin, ConfigPath: filepath.Join(t.TempDir(), "rclone.conf"), Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	bus := eventbus.NewBus(context.Background())
	t.Cleanup(func() { _ = bus.Close() })
	se := syncengine.New(syncengine.Deps{Logger: log, Bus: bus, Store: st, Rclone: rc})
	if err := se.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = se.Stop(context.Background()) })
	fe := New(Options{Store: st, Sync: se, Bus: bus, Log: log})
	return fe, st, se
}

func saveFlow(t *testing.T, st *store.Store, id string) {
	t.Helper()
	f := &store.Flow{
		ID:   id,
		Name: "t",
		Operations: []store.Operation{{
			ID:         "op1",
			FlowID:     id,
			SourcePath: filepath.Join(t.TempDir(), "src"),
			TargetPath: filepath.Join(t.TempDir(), "dst"),
			Action:     "push",
		}},
	}
	if err := st.Flows().Save(context.Background(), f); err != nil {
		t.Fatal(err)
	}
}

func TestExecute_EmptyAndAlreadyRunning(t *testing.T) {
	fe, st, _ := testStack(t, writeSleepRclone(t))
	if err := fe.Execute(context.Background(), "missing"); err == nil {
		t.Fatal("expected missing flow error")
	}
	empty := &store.Flow{ID: "empty", Name: "e"}
	if err := st.Flows().Save(context.Background(), empty); err != nil {
		t.Fatal(err)
	}
	if err := fe.Execute(context.Background(), "empty"); err != ErrEmptyFlow {
		t.Fatalf("empty: %v", err)
	}

	saveFlow(t, st, "f1")
	if err := fe.Execute(context.Background(), "f1"); err != nil {
		t.Fatal(err)
	}
	if !fe.IsRunning("f1") {
		t.Fatal("expected running")
	}
	if err := fe.Execute(context.Background(), "f1"); err != ErrAlreadyRunning {
		t.Fatalf("second execute: %v", err)
	}
	if err := fe.Stop("f1"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !fe.IsRunning("f1") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if fe.IsRunning("f1") {
		t.Fatal("still running after stop")
	}
	if fe.Status("f1") != "cancelled" {
		t.Fatalf("status = %q, want cancelled", fe.Status("f1"))
	}
}

func TestExecute_Completes(t *testing.T) {
	fe, st, _ := testStack(t, writeOkRclone(t))
	saveFlow(t, st, "f2")
	if err := fe.Execute(context.Background(), "f2"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !fe.IsRunning("f2") && fe.Status("f2") == "completed" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("status = %q running=%v", fe.Status("f2"), fe.IsRunning("f2"))
}

func TestStop_NotRunning(t *testing.T) {
	fe, _, _ := testStack(t, writeOkRclone(t))
	if err := fe.Stop("nope"); err != ErrNotRunning {
		t.Fatalf("got %v", err)
	}
}
