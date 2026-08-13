// Package runtimehub is the in-process projection of live run state.
//
// The Go process owns both persisted documents (SQLite) and runtime
// (flow/op status, transfers, last 40 log lines). This hub listens to
// flow:execution and sync:* events, seeds from flowengine.Statuses and
// syncengine.ActiveTasks, and serves GET /api/v1/runtime plus the first
// SSE frame (runtime:snapshot). The SPA hydrates Pinia from that
// snapshot and then applies live events; it does not invent runtime.
// Runtime is memory-only: a process restart drops it. A page reload
// does not.
package runtimehub

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gnasdev/gn-drive/internal/eventbus"
	"github.com/gnasdev/gn-drive/internal/rclone"
	"github.com/gnasdev/gn-drive/internal/syncengine"
)

const maxLog = 40

// FlowStatusSource is satisfied by flowengine.Engine.
type FlowStatusSource interface {
	Statuses() map[string]string
}

// TaskSource is satisfied by syncengine.Engine.
type TaskSource interface {
	ActiveTasks(ctx context.Context) ([]syncengine.TaskSnapshot, error)
}

// Options wires the hub to the process event bus and optional engines
// used to seed a snapshot if events were missed.
type Options struct {
	Bus   *eventbus.Bus
	Flows FlowStatusSource
	Tasks TaskSource
}

type opRuntime struct {
	Status    string
	LastError string
}

type flowRuntime struct {
	Status    string
	LastError string
	Ops       map[string]opRuntime
	Sync      *eventbus.SyncProgressEvent
	Log       []eventbus.RuntimeLogEntry
}

// Hub is the in-memory runtime projection. It is not persisted.
type Hub struct {
	mu     sync.Mutex
	rev    int64
	flows  map[string]*flowRuntime
	flowsS FlowStatusSource
	tasks  TaskSource
	cancel func()
}

// New subscribes to flow/sync topics. Close unsubscribes.
func New(opts Options) *Hub {
	h := &Hub{
		flows:  make(map[string]*flowRuntime),
		flowsS: opts.Flows,
		tasks:  opts.Tasks,
	}
	if opts.Bus != nil {
		h.cancel = opts.Bus.SubscribeAll([]string{
			eventbus.TopicFlowExecution,
			eventbus.TopicSyncStarted,
			eventbus.TopicSyncProgress,
			eventbus.TopicSyncCompleted,
			eventbus.TopicSyncFailed,
		}, h.onEvent)
	}
	return h
}

// Close stops bus subscriptions. Safe on a nil hub.
func (h *Hub) Close() {
	if h == nil {
		return
	}
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}
}

// Reset clears the projection (lock / data-plane teardown).
func (h *Hub) Reset() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.flows = make(map[string]*flowRuntime)
	h.rev++
}

// Forget drops a deleted flow from the projection.
func (h *Hub) Forget(flowID string) {
	if h == nil || flowID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.flows, flowID)
	h.rev++
}

// Snapshot returns a copy of the projection, overlaid with current engine
// statuses and active rclone tasks so a late subscriber still sees a live run.
func (h *Hub) Snapshot() eventbus.RuntimeSnapshotEvent {
	if h == nil {
		return eventbus.NewRuntimeSnapshot(0, nil)
	}
	h.mu.Lock()
	rev := h.rev
	cloned := cloneFlows(h.flows)
	h.mu.Unlock()

	if h.flowsS != nil {
		for id, st := range h.flowsS.Statuses() {
			fr := cloned[id]
			if fr == nil {
				fr = &flowRuntime{Ops: map[string]opRuntime{}}
				cloned[id] = fr
			}
			if st != "" {
				fr.Status = st
			}
		}
	}
	if h.tasks != nil {
		tasks, err := h.tasks.ActiveTasks(context.Background())
		if err == nil {
			for _, t := range tasks {
				applyTask(cloned, t)
			}
		}
	}

	out := make([]eventbus.RuntimeFlowState, 0, len(cloned))
	for id, fr := range cloned {
		out = append(out, toFlowState(id, fr))
	}
	return eventbus.NewRuntimeSnapshot(rev, out)
}

func (h *Hub) onEvent(topic string, ev eventbus.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch topic {
	case eventbus.TopicFlowExecution:
		fe, ok := ev.(eventbus.FlowExecutionEvent)
		if !ok || fe.FlowID == "" {
			return
		}
		h.applyFlowLocked(fe)
	case eventbus.TopicSyncStarted, eventbus.TopicSyncProgress, eventbus.TopicSyncCompleted, eventbus.TopicSyncFailed:
		h.applySyncLocked(topic, ev)
	default:
		return
	}
	h.rev++
}

func (h *Hub) applyFlowLocked(fe eventbus.FlowExecutionEvent) {
	fr := h.ensureLocked(fe.FlowID)
	if fe.OpID != "" {
		op := fr.Ops[fe.OpID]
		op.Status = fe.Status
		if fe.Error != "" {
			op.LastError = fe.Error
			if fe.Status != "cancelled" && fe.Status != "cancelling" {
				fr.LastError = fe.Error
			}
		}
		fr.Ops[fe.OpID] = op
		if fe.Status == "running" || fe.Status == "cancelling" || fe.Status == "failed" || fe.Status == "cancelled" {
			fr.Status = fe.Status
		}
	} else {
		fr.Status = fe.Status
		if fe.Error != "" && fe.Status != "cancelled" && fe.Status != "cancelling" {
			fr.LastError = fe.Error
		}
		if fe.Status == "cancelled" || fe.Status == "cancelling" {
			fr.LastError = ""
		}
	}
	label := "Flow"
	if fe.OpID != "" {
		label = "Op " + shortID(fe.OpID)
	}
	fr.Log = appendLog(fr.Log, eventbus.RuntimeLogEntry{
		At:     time.Now().UnixMilli(),
		Status: fe.Status,
		OpID:   fe.OpID,
		Error:  fe.Error,
		Label:  label,
	})
}

func (h *Hub) applySyncLocked(topic string, ev eventbus.Event) {
	prog, ok := syncEvent(topic, ev)
	if !ok {
		return
	}
	flowID, opID, ok := splitBusyKey(prog.ProfileID)
	if !ok {
		return
	}
	fr := h.ensureLocked(flowID)
	cp := prog
	fr.Sync = &cp
	if opID != "" {
		op := fr.Ops[opID]
		if prog.State != "" {
			op.Status = prog.State
		} else if topic == eventbus.TopicSyncStarted {
			op.Status = "running"
		}
		if prog.ErrorMessage != "" {
			op.LastError = prog.ErrorMessage
		}
		fr.Ops[opID] = op
	}
	switch {
	case topic == eventbus.TopicSyncStarted:
		fr.Status = "running"
	case prog.State == "running":
		fr.Status = "running"
	case prog.State == "failed" || prog.State == "cancelled" || prog.State == "completed":
		// Flow-level terminal status comes from flow:execution (no op id).
		// Keep running until that arrives unless this is the only signal.
		if fr.Status == "" {
			fr.Status = prog.State
		}
	}
	if prog.ErrorMessage != "" && prog.State != "cancelled" {
		fr.LastError = prog.ErrorMessage
	}
}

func (h *Hub) ensureLocked(flowID string) *flowRuntime {
	fr := h.flows[flowID]
	if fr == nil {
		fr = &flowRuntime{Ops: make(map[string]opRuntime)}
		h.flows[flowID] = fr
	}
	if fr.Ops == nil {
		fr.Ops = make(map[string]opRuntime)
	}
	return fr
}

func syncEvent(topic string, ev eventbus.Event) (eventbus.SyncProgressEvent, bool) {
	switch topic {
	case eventbus.TopicSyncStarted:
		if st, ok := ev.(eventbus.SyncStartedEvent); ok {
			return eventbus.SyncProgressEvent{
				TaskID:    st.TaskID,
				ProfileID: st.ProfileID,
				Action:    st.Action,
				State:     "running",
			}, true
		}
	case eventbus.TopicSyncProgress, eventbus.TopicSyncFailed:
		if p, ok := ev.(eventbus.SyncProgressEvent); ok {
			return p, true
		}
	case eventbus.TopicSyncCompleted:
		if c, ok := ev.(eventbus.SyncCompletedEvent); ok {
			return eventbus.SyncProgressEvent{
				TaskID:           c.TaskID,
				ProfileID:        c.ProfileID,
				Action:           c.Action,
				State:            "completed",
				Transferred:      c.Bytes,
				FilesTransferred: 0,
				Errors:           c.Errors,
			}, true
		}
	}
	return eventbus.SyncProgressEvent{}, false
}

func applyTask(dst map[string]*flowRuntime, t syncengine.TaskSnapshot) {
	flowID, opID, ok := splitBusyKey(t.Name)
	if !ok {
		return
	}
	fr := dst[flowID]
	if fr == nil {
		fr = &flowRuntime{Ops: map[string]opRuntime{}}
		dst[flowID] = fr
	}
	if fr.Ops == nil {
		fr.Ops = map[string]opRuntime{}
	}
	status := t.Status
	if status == "" {
		status = "running"
	}
	fr.Status = status
	if opID != "" {
		fr.Ops[opID] = opRuntime{Status: status}
	}
	prog := progressFromTask(t)
	fr.Sync = &prog
}

func progressFromTask(t syncengine.TaskSnapshot) eventbus.SyncProgressEvent {
	s := t.Stats
	return eventbus.SyncProgressEvent{
		TaskID:           t.ID,
		ProfileID:        t.Name,
		Action:           t.Action,
		State:            orRunning(t.Status),
		Transferred:      s.Bytes,
		Total:            s.BytesTotal,
		BytesPerSec:      s.Speed,
		ETA:              s.ETA,
		Errors:           int(s.Errors),
		CurrentFile:      s.CurrentFile,
		FilesTransferred: int(s.Files),
		TotalFiles:       int(s.FilesTotal),
		Checks:           s.Checks,
		TotalChecks:      s.ChecksTotal,
		Deletes:          s.Deletes,
		Renames:          s.Renames,
		Transfers:        transfersFromStats(s),
	}
}

func transfersFromStats(s rclone.Stats) []eventbus.FileTransferEvent {
	out := make([]eventbus.FileTransferEvent, 0, len(s.FileTransfers))
	for _, ft := range s.FileTransfers {
		out = append(out, eventbus.FileTransferEvent{
			Name:     ft.Name,
			Size:     ft.Size,
			Bytes:    ft.Bytes,
			Progress: ft.Progress,
			Status:   ft.Status,
			Speed:    ft.Speed,
			Error:    ft.Error,
		})
	}
	return out
}

func toFlowState(id string, fr *flowRuntime) eventbus.RuntimeFlowState {
	ops := make([]eventbus.RuntimeOpState, 0, len(fr.Ops))
	for opID, op := range fr.Ops {
		ops = append(ops, eventbus.RuntimeOpState{
			ID:        opID,
			Status:    op.Status,
			LastError: op.LastError,
		})
	}
	log := append([]eventbus.RuntimeLogEntry(nil), fr.Log...)
	var sync *eventbus.SyncProgressEvent
	if fr.Sync != nil {
		cp := *fr.Sync
		sync = &cp
	}
	status := fr.Status
	if status == "" {
		status = "idle"
	}
	return eventbus.RuntimeFlowState{
		ID:        id,
		Status:    status,
		LastError: fr.LastError,
		Ops:       ops,
		Sync:      sync,
		Log:       log,
	}
}

func cloneFlows(in map[string]*flowRuntime) map[string]*flowRuntime {
	out := make(map[string]*flowRuntime, len(in))
	for id, fr := range in {
		cp := &flowRuntime{
			Status:    fr.Status,
			LastError: fr.LastError,
			Ops:       make(map[string]opRuntime, len(fr.Ops)),
			Log:       append([]eventbus.RuntimeLogEntry(nil), fr.Log...),
		}
		for k, v := range fr.Ops {
			cp.Ops[k] = v
		}
		if fr.Sync != nil {
			s := *fr.Sync
			if s.Transfers != nil {
				s.Transfers = append([]eventbus.FileTransferEvent(nil), s.Transfers...)
			}
			cp.Sync = &s
		}
		out[id] = cp
	}
	return out
}

func appendLog(prev []eventbus.RuntimeLogEntry, e eventbus.RuntimeLogEntry) []eventbus.RuntimeLogEntry {
	if n := len(prev); n > 0 {
		last := prev[n-1]
		if last.Status == e.Status && last.OpID == e.OpID && last.Error == e.Error {
			return prev
		}
	}
	next := append(prev, e)
	if len(next) > maxLog {
		next = append([]eventbus.RuntimeLogEntry(nil), next[len(next)-maxLog:]...)
	}
	return next
}

func splitBusyKey(profileID string) (flowID, opID string, ok bool) {
	i := strings.IndexByte(profileID, ':')
	if i <= 0 || i >= len(profileID)-1 {
		return "", "", false
	}
	return profileID[:i], profileID[i+1:], true
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func orRunning(s string) string {
	if s == "" {
		return "running"
	}
	return s
}
