package cqrshtmx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-sse"
)

// TestNotificationLevelString pins the String round-trip on all levels plus
// an arbitrary value (the type is a plain string alias, so String must never
// transform).
func TestNotificationLevelString(t *testing.T) {
	t.Parallel()

	cases := []NotificationLevel{
		LevelSuccess,
		LevelError,
		LevelWarning,
		LevelInfo,
		NotificationLevel("custom"),
	}
	for _, level := range cases {
		if got := level.String(); got != string(level) {
			t.Errorf("NotificationLevel(%q).String() = %q", level, got)
		}
	}
}

// TestDelegatingWriterUnwrap proves http.ResponseController can reach the
// underlying writer through the wrapper chain — the reason Unwrap exists.
func TestDelegatingWriterUnwrap(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	wrapped := newDelegatingWriter(rec)

	if unwrapped := wrapped.Unwrap(); unwrapped != rec {
		t.Error("Unwrap did not return the wrapped recorder")
	}

	// ResponseController locates Flusher through the Unwrap chain; a panic or
	// dead Flush here means the delegation contract broke.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Flush through ResponseController panicked: %v", r)
			}
		}()

		http.NewResponseController(wrapped).Flush()
	}()
}

// TestParseSSEEventID covers the deprecated delegation shims: valid parse,
// newline rejection, must-variant round-trip, and the wire-format writer.
func TestParseSSEEventID(t *testing.T) {
	t.Parallel()

	id, err := ParseSSEEventID("evt-1")
	if err != nil {
		t.Fatalf("ParseSSEEventID(valid): %v", err)
	}

	if id != MustParseSSEEventID("evt-1") {
		t.Error("ParseSSEEventID and MustParseSSEEventID disagree")
	}

	if _, err := ParseSSEEventID("bad\nid"); err == nil {
		t.Error("ParseSSEEventID accepted a newline")
	}

	if got := MustParseSSEEventID("evt-2"); got != MustParseSSEEventID("evt-2") {
		t.Error("MustParseSSEEventID is not deterministic")
	}

	if NewSSEEventID("evt-3") != MustParseSSEEventID("evt-3") {
		t.Error("NewSSEEventID and MustParseSSEEventID disagree")
	}
}

// TestWriteSSEEventAndHeaders pins the deprecated wire-format writer and the
// SSE response headers.
func TestWriteSSEEventAndHeaders(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	evt := SSEEvent{Event: "domain", Data: `{"k":"v"}`, ID: sse.NewEventID("evt-9")}
	if err := WriteSSEEvent(&buf, evt); err != nil {
		t.Fatalf("WriteSSEEvent: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "event: domain") || !strings.Contains(out, `{"k":"v"}`) {
		t.Errorf("wire format missing fields: %q", out)
	}

	if !strings.Contains(out, "id: evt-9") {
		t.Errorf("wire format missing id: %q", out)
	}

	rec := httptest.NewRecorder()
	SetSSEHeaders(rec)

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
}

// TestWithMaxReplayLimitsFirstConnect proves the deprecated option still
// clamps the zero-cursor replay on the journal SSE store.
func TestWithMaxReplayLimitsFirstConnect(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 4)
	appendSyncEvents(t, store, events)

	mapper := func(evt event.Event) sse.Event {
		return sse.Event{Event: "domain", Data: "x"}
	}
	journal := NewJournalSSEStore(store, mapper, WithMaxReplay(1))

	var zeroID sse.EventID

	got, err := journal.EventsAfter(zeroID)
	if err != nil {
		t.Fatalf("EventsAfter: %v", err)
	}

	if len(got) != 1 {
		t.Errorf("replay = %d events, want clamped to 1", len(got))
	}
}

// TestWithSyncPullLimitDefaultAndExplicitOverride pins the documented
// contract: the option is the DEFAULT batch size; an explicit ?limit=
// overrides it (clamped only by the global MaxSyncPullLimit) — a prior doc
// comment claimed the option was also a ceiling, which the handler never
// enforced.
func TestWithSyncPullLimitDefaultAndCeiling(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	events := seedSyncEvents(t, 5)
	appendSyncEvents(t, store, events)

	handler := SyncPullHandler(store, WithSyncPullLimit(2))

	rec := doPull(handler, "/sync/pull")
	if rec.Code != http.StatusOK {
		t.Fatalf("default-limit status = %d", rec.Code)
	}

	if resp := pullSyncBody(t, rec); len(resp.Events) != 2 {
		t.Errorf("default pull = %d events, want default 2", len(resp.Events))
	}

	rec = doPull(handler, "/sync/pull?limit=4")
	if resp := pullSyncBody(t, rec); len(resp.Events) != 4 {
		t.Errorf("explicit override = %d events, want 4", len(resp.Events))
	}
}
