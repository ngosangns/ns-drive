// Package eventbus provides in-process typed event channels.
// See bus.go for the Bus interface.
package eventbus

import "time"

// Event is the base interface for all bus events.
type Event interface {
	eventMarker() // prevents accidental interface{} use
}

type eventBase struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

func (eventBase) eventMarker() {}

// --- Sync events -----------------------------------------------------------

// FileTransferEvent is one file row in SyncProgressEvent.Transfers (Wails FileTransferInfo).
type FileTransferEvent struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Bytes    int64   `json:"bytes"`
	Progress float64 `json:"progress"`
	Status   string  `json:"status"` // transferring | completed | failed | checking | checked | pending
	Speed    float64 `json:"speed,omitempty"`
	Error    string  `json:"error,omitempty"`
}

// SyncProgressEvent is emitted periodically during a sync task.
// Field set mirrors Wails SyncStatusDTO so the web status panel can match desktop.
type SyncProgressEvent struct {
	eventBase
	TaskID           string  `json:"task_id"`
	ProfileID        string  `json:"profile_id"`
	Action           string  `json:"action"`
	State            string  `json:"state"` // running, completed, failed, cancelled
	Transferred      int64   `json:"transferred"`
	Total            int64   `json:"total"`
	BytesPerSec      float64 `json:"bytes_per_sec"`
	ETA              int64   `json:"eta_secs"`
	Errors           int     `json:"errors"`
	CurrentFile      string  `json:"current_file"`
	FilesTransferred int     `json:"files_transferred"`
	TotalFiles       int     `json:"total_files"`
	Checks           int64   `json:"checks"`
	TotalChecks      int64   `json:"total_checks"`
	Deletes          int64   `json:"deletes"`
	Renames          int64   `json:"renames"`
	// Transfers is the per-file list for Syncing / Complete / Error / Pending tabs.
	Transfers []FileTransferEvent `json:"transfers,omitempty"`
	// ErrorMessage is set on failed sync events so the UI can surface the
	// reason (omitted for running/completed events).
	ErrorMessage string `json:"error_message,omitempty"`
}

// SyncStartedEvent is emitted when a sync task begins.
type SyncStartedEvent struct {
	eventBase
	TaskID    string `json:"task_id"`
	ProfileID string `json:"profile_id"`
	Action    string `json:"action"`
}

// SyncCompletedEvent is emitted when a sync task finishes.
type SyncCompletedEvent struct {
	eventBase
	TaskID    string `json:"task_id"`
	ProfileID string `json:"profile_id"`
	Action    string `json:"action"`
	Duration  int64  `json:"duration_secs"`
	Bytes     int64  `json:"bytes"`
	Errors    int    `json:"errors"`
}

// --- Auth events -----------------------------------------------------------

// AuthUnlockedEvent is emitted after successful master password unlock.
type AuthUnlockedEvent struct {
	eventBase
}

// AuthLockedEvent is emitted after lock.
type AuthLockedEvent struct {
	eventBase
}

// --- Service events --------------------------------------------------------

// ServiceStatusEvent is emitted on service state changes.
type ServiceStatusEvent struct {
	eventBase
	Running    bool `json:"running"`
	WebPort    int  `json:"web_port"`
	UptimeSecs int  `json:"uptime_secs"`
}

// --- Schedule events -------------------------------------------------------

// ScheduleTriggeredEvent is emitted when a cron schedule fires.
type ScheduleTriggeredEvent struct {
	eventBase
	ScheduleID string `json:"schedule_id"`
	ProfileID  string `json:"profile_id"`
	Action     string `json:"action"`
}

// --- Board events ----------------------------------------------------------

// BoardExecutionEvent is emitted during board DAG execution.
// Flow engine also emits this (board_id = flow id) for backward compatibility.
type BoardExecutionEvent struct {
	eventBase
	BoardID   string `json:"board_id"`
	NodeID    string `json:"node_id,omitempty"`
	EdgeID    string `json:"edge_id,omitempty"`
	Status    string `json:"status"` // running, completed, failed
	ProfileID string `json:"profile_id,omitempty"`
	Action    string `json:"action,omitempty"`
}

// FlowExecutionEvent is emitted during sequential flow operation runs.
type FlowExecutionEvent struct {
	eventBase
	FlowID string `json:"flow_id"`
	OpID   string `json:"op_id,omitempty"`
	Status string `json:"status"` // running, completed, failed, cancelled, cancelling
	Error  string `json:"error,omitempty"`
}

// --- State events ----------------------------------------------------------

// StateChangedEvent is emitted when persisted document state changes.
// Domain is flows | remotes | profiles | settings. ID is the entity when known.
type StateChangedEvent struct {
	eventBase
	Domain string `json:"domain"`
	ID     string `json:"id,omitempty"`
}

// --- Runtime snapshot ------------------------------------------------------

// RuntimeLogEntry is one line of a flow run log (mirrors the SPA panel).
type RuntimeLogEntry struct {
	At     int64  `json:"at"`
	Status string `json:"status"`
	OpID   string `json:"op_id,omitempty"`
	Error  string `json:"error,omitempty"`
	Label  string `json:"label,omitempty"`
}

// RuntimeOpState is the last known lifecycle status of one operation.
type RuntimeOpState struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	LastError string `json:"last_error,omitempty"`
}

// RuntimeFlowState is the backend-owned runtime view of one flow.
type RuntimeFlowState struct {
	ID        string             `json:"id"`
	Status    string             `json:"status"`
	LastError string             `json:"last_error,omitempty"`
	Ops       []RuntimeOpState   `json:"ops,omitempty"`
	Sync      *SyncProgressEvent `json:"sync,omitempty"`
	Log       []RuntimeLogEntry  `json:"log,omitempty"`
}

// RuntimeSnapshotEvent is the full runtime projection. Sent as the first SSE
// frame on connect and as GET /api/v1/runtime so a reload can hydrate
// without waiting for the next live tick.
type RuntimeSnapshotEvent struct {
	eventBase
	Revision int64              `json:"revision"`
	Flows    []RuntimeFlowState `json:"flows"`
}

// NewRuntimeSnapshot builds a snapshot event with type/timestamp set.
func NewRuntimeSnapshot(revision int64, flows []RuntimeFlowState) RuntimeSnapshotEvent {
	if flows == nil {
		flows = []RuntimeFlowState{}
	}
	return RuntimeSnapshotEvent{
		eventBase: eventBase{Type: TopicRuntimeSnapshot, Timestamp: time.Now()},
		Revision:  revision,
		Flows:     flows,
	}
}
