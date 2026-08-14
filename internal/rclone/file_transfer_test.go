package rclone

import (
	"fmt"
	"testing"
)

func TestFileTransferTracker_SeedPendingAndPromote(t *testing.T) {
	tr := newFileTransferTracker()
	tr.seedPending([]FileEntry{
		{Name: "a.txt", Path: "a.txt", Size: 10},
		{Name: "b.txt", Path: "b.txt", Size: 20},
		{Name: "dir", Path: "dir", IsDir: true}, // skipped
	})

	snap := tr.snapshot(0)
	if len(snap) != 2 {
		t.Fatalf("seeded len = %d, want 2", len(snap))
	}
	for _, ft := range snap {
		if ft.Status != "pending" {
			t.Errorf("%s status = %q, want pending", ft.Name, ft.Status)
		}
	}

	// Promote one file to transferring — must not stay pending.
	tr.upsert(FileTransfer{Name: "a.txt", Size: 10, Bytes: 5, Progress: 50, Status: "transferring"})
	snap = tr.snapshot(0)
	byName := map[string]FileTransfer{}
	for _, ft := range snap {
		byName[ft.Name] = ft
	}
	if byName["a.txt"].Status != "transferring" {
		t.Errorf("a.txt status = %q, want transferring", byName["a.txt"].Status)
	}
	if byName["b.txt"].Status != "pending" {
		t.Fatalf("b.txt status = %q, want pending", byName["b.txt"].Status)
	}

	// Seed again must not demote transferring back to pending.
	tr.seedPending([]FileEntry{{Name: "a.txt", Path: "a.txt", Size: 10}})
	if tr.byName["a.txt"].Status != "transferring" {
		t.Fatalf("re-seed demoted a.txt to %q", tr.byName["a.txt"].Status)
	}
}

func TestFileTransferTracker_SnapshotSyntheticPending(t *testing.T) {
	tr := newFileTransferTracker()
	tr.upsert(FileTransfer{Name: "done.txt", Status: "completed", Progress: 100})
	// totalFiles=3, known completed=1 → synthetic "(2 pending)"
	snap := tr.snapshot(3)
	var synth *FileTransfer
	for i := range snap {
		if snap[i].Status == "pending" {
			synth = &snap[i]
			break
		}
	}
	if synth == nil || synth.Name != "(2 pending)" {
		t.Fatalf("synthetic pending = %+v, want (2 pending)", synth)
	}

	// Named pending counts toward known — no extra synthetic.
	tr.seedPending([]FileEntry{{Path: "x.bin", Size: 1}, {Path: "y.bin", Size: 1}})
	snap = tr.snapshot(3) // completed + 2 pending named = 3
	for _, ft := range snap {
		if ft.Name == "(2 pending)" || (len(ft.Name) > 0 && ft.Name[0] == '(') {
			t.Fatalf("unexpected synthetic when named pending cover total: %+v", ft)
		}
	}
}

// TestFileTransferTracker_SeedNeverEvictsLiveProgress reproduces a bug where
// a bulk seedPending call (which runs concurrently with the transfer and can
// land after real progress already arrived) evicted completed/transferring
// rows to make room for cosmetic pending placeholders, hiding real progress
// from the UI.
func TestFileTransferTracker_SeedNeverEvictsLiveProgress(t *testing.T) {
	tr := newFileTransferTracker()
	// Real progress arrives first, as it does when transfers start before the
	// (slow, recursive) pending listing finishes.
	tr.upsert(FileTransfer{Name: "done.txt", Status: "completed", Progress: 100})
	tr.upsert(FileTransfer{Name: "active.txt", Status: "transferring", Progress: 40})

	// Fill the tracker with pending placeholders past capacity.
	entries := make([]FileEntry, 0, maxTrackedFiles+10)
	for i := 0; i < maxTrackedFiles+10; i++ {
		entries = append(entries, FileEntry{Path: fmt.Sprintf("seed-%d.bin", i), Size: 1})
	}
	tr.seedPending(entries)

	if tr.byName["done.txt"] == nil || tr.byName["done.txt"].Status != "completed" {
		t.Fatalf("done.txt evicted or demoted by pending seed: %+v", tr.byName["done.txt"])
	}
	if tr.byName["active.txt"] == nil || tr.byName["active.txt"].Status != "transferring" {
		t.Fatalf("active.txt evicted or demoted by pending seed: %+v", tr.byName["active.txt"])
	}
}

func TestPendingSeedPath(t *testing.T) {
	push := pendingSeedPath(SyncConfig{Action: ActionPush, Source: "/src", Dest: "s3:bucket"})
	if push != "/src" {
		t.Errorf("push seed = %q, want /src", push)
	}
	pull := pendingSeedPath(SyncConfig{Action: ActionPull, Source: "/src", Dest: "s3:bucket"})
	if pull != "s3:bucket" {
		t.Errorf("pull seed = %q, want s3:bucket", pull)
	}
}

func TestIngestJSONLogLine_TransferringAndCompleted(t *testing.T) {
	tr := newFileTransferTracker()
	var s Stats
	// Seed pending first, then promote via log lines.
	tr.seedPending([]FileEntry{{Path: "photo.jpg", Size: 9000}})
	ingestJSONLogLine(
		`{"level":"info","msg":"stats","stats":{"bytes":100,"totalBytes":9000,"transfers":0,"totalTransfers":1,"transferring":[{"name":"photo.jpg","size":9000,"bytes":100,"percentage":1,"speed":50}]}}`,
		&s, tr,
	)
	if tr.byName["photo.jpg"].Status != "transferring" {
		t.Fatalf("status after transferring = %q", tr.byName["photo.jpg"].Status)
	}
	ingestJSONLogLine(
		`{"level":"info","msg":"Copied (new)","object":"photo.jpg","size":9000}`,
		&s, tr,
	)
	if tr.byName["photo.jpg"].Status != "completed" {
		t.Fatalf("status after copied = %q", tr.byName["photo.jpg"].Status)
	}
	snap := tr.snapshot(1)
	if len(snap) != 1 || snap[0].Status != "completed" {
		t.Fatalf("final snap = %+v", snap)
	}
}
