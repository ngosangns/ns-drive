package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/gnasdev/gn-drive/internal/eventbus"
	"github.com/gnasdev/gn-drive/internal/rclone"
	"github.com/gnasdev/gn-drive/internal/store"
)

const stateSocketWriteTimeout = 10 * time.Second

// StateSnapshot is the canonical UI projection sent whenever a state socket
// connects. Persisted documents and live runtime have separate revisions.
type StateSnapshot struct {
	Flows    []store.Flow                  `json:"flows"`
	Remotes  []rclone.Remote               `json:"remotes"`
	Settings map[string]string             `json:"settings"`
	Runtime  eventbus.RuntimeSnapshotEvent `json:"runtime"`
}

type stateFrame struct {
	Type      string          `json:"type"`
	CommandID string          `json:"command_id,omitempty"`
	Topic     string          `json:"topic,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Snapshot  *StateSnapshot  `json:"snapshot,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// handleStateSocket is the authenticated, backend-owned state stream. The
// auth middleware validates the session before the WebSocket handshake.
func (s *Server) handleStateSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()
	var writeMu sync.Mutex
	write := func(frame stateFrame) error {
		payload, err := json.Marshal(frame)
		if err != nil {
			return err
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		writeCtx, cancel := context.WithTimeout(ctx, stateSocketWriteTimeout)
		defer cancel()
		return conn.Write(writeCtx, websocket.MessageText, payload)
	}

	// Subscribe before the first snapshot. If state changes while snapshotting,
	// the queued event follows the snapshot and the client converges.
	var cancel func()
	if s.app != nil && s.app.Bus != nil {
		cancel = s.app.Bus.SubscribeAll(eventbus.AllTopics(), func(topic string, ev eventbus.Event) {
			if topic == eventbus.TopicAuthLocked {
				_ = conn.Close(websocket.StatusPolicyViolation, "application locked")
				return
			}
			if topic != eventbus.TopicStateChanged && topic != eventbus.TopicFlowExecution &&
				topic != eventbus.TopicSyncStarted && topic != eventbus.TopicSyncProgress &&
				topic != eventbus.TopicSyncCompleted && topic != eventbus.TopicSyncFailed {
				return
			}
			if topic == eventbus.TopicFlowExecution || topic == eventbus.TopicSyncStarted ||
				topic == eventbus.TopicSyncProgress || topic == eventbus.TopicSyncCompleted || topic == eventbus.TopicSyncFailed {
				data, marshalErr := json.Marshal(ev)
				if marshalErr == nil {
					_ = write(stateFrame{Type: "runtime.event", Topic: topic, Data: data})
				}
				return
			}
			snap, snapshotErr := s.stateSnapshot(context.Background())
			if snapshotErr == nil {
				_ = write(stateFrame{Type: "state.snapshot", Snapshot: &snap})
			}
		})
	}
	if cancel != nil {
		defer cancel()
	}

	snapshot, err := s.stateSnapshot(ctx)
	if err != nil {
		_ = write(stateFrame{Type: "state.error", Error: err.Error()})
		return
	}
	if err := write(stateFrame{Type: "state.snapshot", Snapshot: &snapshot}); err != nil {
		return
	}

	for {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var frame stateFrame
		if json.Unmarshal(raw, &frame) != nil || frame.Type != "state.resync" {
			_ = write(stateFrame{Type: "state.error", CommandID: frame.CommandID, Error: "unsupported state command"})
			continue
		}
		snapshot, err := s.stateSnapshot(ctx)
		if err != nil {
			_ = write(stateFrame{Type: "state.error", CommandID: frame.CommandID, Error: err.Error()})
			continue
		}
		_ = write(stateFrame{Type: "state.snapshot", CommandID: frame.CommandID, Snapshot: &snapshot})
	}
}

func (s *Server) stateSnapshot(ctx context.Context) (StateSnapshot, error) {
	if s.app == nil || s.app.Store == nil || s.app.Rclone == nil {
		return StateSnapshot{}, errors.New("data plane not ready")
	}
	flows, err := s.app.Store.Flows().List(ctx)
	if err != nil {
		return StateSnapshot{}, err
	}
	remotes, err := s.app.Rclone.ListRemotes(ctx)
	if err != nil {
		return StateSnapshot{}, err
	}
	if remotes == nil {
		remotes = []rclone.Remote{}
	}
	settings := make(map[string]string)
	for _, key := range []string{"theme", "notifications_enabled", "debug_mode"} {
		if value, getErr := s.app.Store.Settings().Get(ctx, key); getErr == nil {
			settings[key] = value
		}
	}
	return StateSnapshot{Flows: flows, Remotes: remotes, Settings: settings, Runtime: s.runtimeSnapshot()}, nil
}
