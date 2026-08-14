package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/gnasdev/gn-drive/internal/eventbus"
)

func TestStateSocket_SendsCanonicalSnapshotThenRuntime(t *testing.T) {
	srv, cleanup := newTestServerWithRclone(t, writeFakeRclone(t))
	defer cleanup()

	ts := httptest.NewServer(srv.router)
	defer ts.Close()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/state"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, raw, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var first stateFrame
	if err := json.Unmarshal(raw, &first); err != nil {
		t.Fatal(err)
	}
	if first.Type != "state.snapshot" || first.Snapshot == nil {
		t.Fatalf("first frame = %+v, want state.snapshot", first)
	}
	if first.Snapshot.Flows == nil || first.Snapshot.Remotes == nil {
		t.Fatalf("snapshot collections must be non-nil: %+v", first.Snapshot)
	}

	srv.app.Bus.Publish(eventbus.TopicFlowExecution, eventbus.FlowExecutionEvent{
		FlowID: "flow-live", Status: "running",
	})
	_, raw, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var second stateFrame
	if err := json.Unmarshal(raw, &second); err != nil {
		t.Fatal(err)
	}
	if second.Type != "runtime.event" || second.Topic != eventbus.TopicFlowExecution {
		t.Fatalf("second frame = %+v, want flow runtime.event", second)
	}
	var event eventbus.FlowExecutionEvent
	if err := json.Unmarshal(second.Data, &event); err != nil {
		t.Fatal(err)
	}
	if event.FlowID != "flow-live" || event.Status != "running" {
		t.Fatalf("event = %+v", event)
	}

	srv.app.Bus.Publish(eventbus.TopicSyncProgress, eventbus.SyncProgressEvent{
		ProfileID: "flow-live:op-live",
		State:     "running",
		Stage:     "listing",
		Transfers: []eventbus.FileTransferEvent{{
			Name: "uploading.bin", Status: "transferring", Progress: 42,
		}},
	})
	_, raw, err = conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var third stateFrame
	if err := json.Unmarshal(raw, &third); err != nil {
		t.Fatal(err)
	}
	if third.Type != "runtime.event" || third.Topic != eventbus.TopicSyncProgress {
		t.Fatalf("third frame = %+v, want sync runtime.event", third)
	}
	var progress eventbus.SyncProgressEvent
	if err := json.Unmarshal(third.Data, &progress); err != nil {
		t.Fatal(err)
	}
	if progress.Stage != "listing" || len(progress.Transfers) != 1 || progress.Transfers[0].Name != "uploading.bin" {
		t.Fatalf("transfers = %+v", progress.Transfers)
	}
}
