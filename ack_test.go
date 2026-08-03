package cqrshtmx

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-sse"
)

// errTestDispatch is a test-only dispatch error using event-family (no stdlib).
var errTestDispatch = errorfamily.NewRejection("test.dispatch_failed", "dispatch failed")

func TestCommandAck_JSON(t *testing.T) {
	t.Parallel()

	ack := CommandAck{
		CommandID: "test-123",
		Status:    AckConfirmed,
	}

	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("produced invalid JSON: %v", err)
	}

	if parsed["commandId"] != "test-123" {
		t.Errorf("expected commandId 'test-123', got %v", parsed["commandId"])
	}

	if parsed["status"] != "confirmed" {
		t.Errorf("expected status 'confirmed', got %v", parsed["status"])
	}

	if _, hasError := parsed["error"]; hasError {
		t.Errorf("expected no error field for confirmed ack, got %v", parsed["error"])
	}
}

func TestCommandAck_JSONRejected(t *testing.T) {
	t.Parallel()

	ack := CommandAck{
		CommandID: "test-456",
		Status:    AckRejected,
		Error:     "email already exists",
	}

	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("produced invalid JSON: %v", err)
	}

	if parsed["status"] != "rejected" {
		t.Errorf("expected status 'rejected', got %v", parsed["status"])
	}

	if parsed["error"] != "email already exists" {
		t.Errorf("expected error 'email already exists', got %v", parsed["error"])
	}
}

func TestCommandIDFromRequest(t *testing.T) {
	t.Parallel()
	t.Run("present", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
		r.Header.Set(CommandIDHeader, "cmd-abc")

		if id := CommandIDFromRequest(r); id != "cmd-abc" {
			t.Errorf("expected 'cmd-abc', got %q", id)
		}
	})

	t.Run("absent", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
		if id := CommandIDFromRequest(r); id != "" {
			t.Errorf("expected empty, got %q", id)
		}
	})
}

func TestAckStatus_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status AckStatus
		want   bool
	}{
		{AckConfirmed, true},
		{AckRejected, true},
		{AckStatus("confirmed"), true},
		{AckStatus("rejected"), true},
		{AckStatus(""), false},
		{AckStatus("pending"), false},
		{AckStatus("CONFIRMED"), false},
		{AckStatus("ok"), false},
		{AckStatus("success"), false},
	}
	for _, tt := range cases {
		if got := tt.status.Valid(); got != tt.want {
			t.Errorf("AckStatus(%q).Valid() = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestBroadcastOnAck_Success(t *testing.T) {
	t.Parallel()

	broadcaster := NewBroadcaster()

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	hook := broadcaster.BroadcastOnAck()

	r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
	r.Header.Set(CommandIDHeader, "cmd-test-1")

	hook(context.Background(), r, nil)

	evt, ok := <-ch
	if !ok {
		t.Fatal("channel closed without receiving event")
	}

	if evt.Event != "sync:ack" {
		t.Errorf("expected event 'sync:ack', got %q", evt.Event)
	}

	var ack CommandAck
	if err := json.Unmarshal([]byte(evt.Data), &ack); err != nil {
		t.Fatalf("invalid ack JSON: %v", err)
	}

	if ack.CommandID != "cmd-test-1" {
		t.Errorf("expected commandId 'cmd-test-1', got %q", ack.CommandID)
	}

	if ack.Status != AckConfirmed {
		t.Errorf("expected status 'confirmed', got %q", ack.Status)
	}
}

func TestBroadcastOnAck_Rejected(t *testing.T) {
	t.Parallel()

	broadcaster := NewBroadcaster()

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	hook := broadcaster.BroadcastOnAck()

	r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
	r.Header.Set(CommandIDHeader, "cmd-test-2")

	hook(context.Background(), r, errTestDispatch)

	evt, ok := <-ch
	if !ok {
		t.Fatal("channel closed without receiving event")
	}

	var ack CommandAck
	if err := json.Unmarshal([]byte(evt.Data), &ack); err != nil {
		t.Fatalf("invalid ack JSON: %v", err)
	}

	if ack.Status != AckRejected {
		t.Errorf("expected status 'rejected', got %q", ack.Status)
	}

	if ack.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestBroadcastOnAck_NoCommandID(t *testing.T) {
	t.Parallel()

	broadcaster := NewBroadcaster()

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	hook := broadcaster.BroadcastOnAck()

	r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
	hook(context.Background(), r, nil)

	// Channel should have no events (buffered, non-blocking)
	select {
	case evt := <-ch:
		t.Fatalf("expected no broadcast without command ID, got: %+v", evt)
	default:
	}
}

func TestBroadcastOnAckFunc_Custom(t *testing.T) {
	t.Parallel()

	broadcaster := NewBroadcaster()

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	hook := broadcaster.BroadcastOnAckFunc(func(r *http.Request, err error, commandID string) sse.Event {
		return sse.Event{
			Event: "custom:ack",
			Data:  `{"cmd":"` + commandID + `","ok":true}`,
		}
	})

	r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
	r.Header.Set(CommandIDHeader, "cmd-custom-1")

	hook(context.Background(), r, nil)

	evt, ok := <-ch
	if !ok {
		t.Fatal("channel closed without receiving event")
	}

	if evt.Event != "custom:ack" {
		t.Errorf("expected event 'custom:ack', got %q", evt.Event)
	}

	if evt.Data != `{"cmd":"cmd-custom-1","ok":true}` {
		t.Errorf("unexpected data: %q", evt.Data)
	}
}

func TestBroadcastOnAckWS_Success(t *testing.T) {
	t.Parallel()

	broadcaster := NewWSBroadcaster()

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	hook := broadcaster.BroadcastOnAckWS()

	r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
	r.Header.Set(CommandIDHeader, "ws-cmd-1")

	hook(context.Background(), r, nil)

	msg, ok := <-ch
	if !ok {
		t.Fatal("channel closed without receiving message")
	}

	var ack CommandAck
	if err := json.Unmarshal([]byte(msg), &ack); err != nil {
		t.Fatalf("invalid ack JSON: %v", err)
	}

	if ack.CommandID != "ws-cmd-1" {
		t.Errorf("expected commandId 'ws-cmd-1', got %q", ack.CommandID)
	}

	if ack.Status != AckConfirmed {
		t.Errorf("expected status 'confirmed', got %q", ack.Status)
	}
}

func TestBroadcastOnAckWS_NoCommandID(t *testing.T) {
	t.Parallel()

	broadcaster := NewWSBroadcaster()

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	hook := broadcaster.BroadcastOnAckWS()

	r := httptest.NewRequest(http.MethodPost, "/api/cmd", nil)
	hook(context.Background(), r, nil)

	select {
	case msg := <-ch:
		t.Fatalf("expected no broadcast without command ID, got: %s", msg)
	default:
	}
}
