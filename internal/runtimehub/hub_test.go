package runtimehub

import (
	"context"
	"testing"
	"time"

	"github.com/gnasdev/gn-drive/internal/eventbus"
	"github.com/gnasdev/gn-drive/internal/rclone"
	"github.com/gnasdev/gn-drive/internal/syncengine"
)

type stubStatuses map[string]string

func (s stubStatuses) Statuses() map[string]string { return s }

type stubTasks struct {
	tasks []syncengine.TaskSnapshot
}

func (s stubTasks) ActiveTasks(context.Context) ([]syncengine.TaskSnapshot, error) {
	return s.tasks, nil
}

func TestSnapshot_Empty(t *testing.T) {
	h := New(Options{})
	snap := h.Snapshot()
	if snap.Revision != 0 {
		t.Errorf("revision = %d, want 0", snap.Revision)
	}
	if snap.Flows == nil || len(snap.Flows) != 0 {
		t.Errorf("flows = %#v, want empty slice", snap.Flows)
	}
}

func TestHub_FlowExecutionAndSync(t *testing.T) {
	bus := eventbus.NewBus(context.Background())
	t.Cleanup(func() { _ = bus.Close() })
	h := New(Options{Bus: bus})
	t.Cleanup(h.Close)

	bus.Publish(eventbus.TopicFlowExecution, eventbus.FlowExecutionEvent{
		FlowID: "f1", Status: "running",
	})
	bus.Publish(eventbus.TopicFlowExecution, eventbus.FlowExecutionEvent{
		FlowID: "f1", OpID: "op1", Status: "running",
	})
	bus.Publish(eventbus.TopicSyncProgress, eventbus.SyncProgressEvent{
		TaskID:      "t1",
		ProfileID:   "f1:op1",
		Action:      "push",
		State:       "running",
		Transferred: 10,
		Total:       100,
		Transfers: []eventbus.FileTransferEvent{
			{Name: "a.txt", Status: "transferring", Progress: 40},
		},
	})

	deadline := time.Now().Add(2 * time.Second)
	var snap eventbus.RuntimeSnapshotEvent
	for time.Now().Before(deadline) {
		snap = h.Snapshot()
		if snap.Revision >= 3 && len(snap.Flows) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if snap.Revision < 3 {
		t.Fatalf("revision = %d, want >= 3 (events not applied)", snap.Revision)
	}
	if len(snap.Flows) != 1 {
		t.Fatalf("flows = %d, want 1", len(snap.Flows))
	}
	f := snap.Flows[0]
	if f.ID != "f1" || f.Status != "running" {
		t.Errorf("flow = %+v", f)
	}
	if f.Sync == nil || len(f.Sync.Transfers) != 1 || f.Sync.Transfers[0].Name != "a.txt" {
		t.Errorf("sync transfers = %+v", f.Sync)
	}
	if len(f.Log) < 2 {
		t.Errorf("log = %+v, want at least flow+op running", f.Log)
	}
	foundOp := false
	for _, op := range f.Ops {
		if op.ID == "op1" && op.Status == "running" {
			foundOp = true
		}
	}
	if !foundOp {
		t.Errorf("ops = %+v, missing op1 running", f.Ops)
	}
}

func TestSnapshot_SeedsActiveTasksAndStatuses(t *testing.T) {
	h := New(Options{
		Flows: stubStatuses{"f9": "completed"},
		Tasks: stubTasks{tasks: []syncengine.TaskSnapshot{{
			ID:     "task-1",
			Name:   "f2:op2",
			Action: "push",
			Status: "running",
			Stats: rclone.Stats{
				Bytes:      3,
				BytesTotal: 9,
				FileTransfers: []rclone.FileTransfer{
					{Name: "b.bin", Status: "transferring", Progress: 30},
				},
			},
		}}},
	})
	snap := h.Snapshot()
	byID := map[string]eventbus.RuntimeFlowState{}
	for _, f := range snap.Flows {
		byID[f.ID] = f
	}
	if byID["f9"].Status != "completed" {
		t.Errorf("f9 status = %q, want completed", byID["f9"].Status)
	}
	f2 := byID["f2"]
	if f2.Status != "running" {
		t.Errorf("f2 status = %q, want running", f2.Status)
	}
	if f2.Sync == nil || len(f2.Sync.Transfers) != 1 || f2.Sync.Transfers[0].Name != "b.bin" {
		t.Errorf("f2 sync = %+v", f2.Sync)
	}
}

func TestHub_ForgetAndFailed(t *testing.T) {
	bus := eventbus.NewBus(context.Background())
	t.Cleanup(func() { _ = bus.Close() })
	h := New(Options{Bus: bus})
	t.Cleanup(h.Close)

	bus.Publish(eventbus.TopicFlowExecution, eventbus.FlowExecutionEvent{
		FlowID: "gone", Status: "running",
	})
	waitRev(t, h, 1)
	h.Forget("gone")
	if len(h.Snapshot().Flows) != 0 {
		t.Fatalf("expected empty after Forget, got %+v", h.Snapshot().Flows)
	}
	afterForget := h.Snapshot().Revision

	bus.Publish(eventbus.TopicFlowExecution, eventbus.FlowExecutionEvent{
		FlowID: "f1", OpID: "op1", Status: "failed", Error: "boom",
	})
	waitRev(t, h, afterForget+1)
	snap := h.Snapshot()
	if len(snap.Flows) != 1 || snap.Flows[0].Status != "failed" || snap.Flows[0].LastError != "boom" {
		t.Errorf("failed flow = %+v", snap.Flows)
	}
}

func TestHub_Reset(t *testing.T) {
	h := New(Options{})
	h.flows["x"] = &flowRuntime{Status: "running", Ops: map[string]opRuntime{}}
	h.rev = 4
	h.Reset()
	if len(h.Snapshot().Flows) != 0 {
		t.Fatalf("Reset left flows: %+v", h.Snapshot().Flows)
	}
	if h.Snapshot().Revision < 5 {
		t.Fatalf("Reset should bump revision, got %d", h.Snapshot().Revision)
	}
}

func TestSplitBusyKey(t *testing.T) {
	flow, op, ok := splitBusyKey("abc:def")
	if !ok || flow != "abc" || op != "def" {
		t.Errorf("got %q %q %v", flow, op, ok)
	}
	if _, _, ok := splitBusyKey("nocolon"); ok {
		t.Error("expected reject")
	}
}

func waitRev(t *testing.T, h *Hub, min int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.Snapshot().Revision >= min {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("revision still %d, want >= %d", h.Snapshot().Revision, min)
}
