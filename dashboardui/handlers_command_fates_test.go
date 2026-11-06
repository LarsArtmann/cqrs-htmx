package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/templ-components/display"
)

// commandFateEvent builds one commandlifecycle event; payloads encode with
// the recorder's default codec (CBOR), which is exactly what the view must
// decode.
func commandFateEvent(
	t *testing.T,
	eventType event.Type,
	version int,
	payload any,
) event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	evt, err := event.New(
		eventType,
		streamID,
		commandlifecycle.StreamTypeCommandLifecycle,
		event.Version(version),
		payload,
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}

func TestCommandFatesIndexHandler_RowsRender(t *testing.T) {
	when := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

	rejected := commandFateEvent(t, commandlifecycle.TypeRejected, 1, commandlifecycle.RejectedPayload{
		CommandID:   commandlifecycle.CommandKey(id.NewCommandID().String()),
		CommandType: "CreateUser",
		Error:       "email already registered",
		Family:      "rejection",
		ErrorCode:   "user.email_taken",
		Attempt:     1,
		RejectedAt:  when,
	})
	deadLettered := commandFateEvent(t, commandlifecycle.TypeDeadLettered, 1, commandlifecycle.DeadLetteredPayload{
		CommandType:    "SyncPush",
		Error:          "store unreachable",
		Attempts:       3,
		DeadLetteredAt: when,
	})
	completed := commandFateEvent(t, commandlifecycle.TypeCompleted, 1, commandlifecycle.CompletedPayload{
		CommandID:   commandlifecycle.CommandKey(id.NewCommandID().String()),
		CommandType: "RenameTenant",
		CompletedAt: when,
	})

	journal := &fakeSeekableJournal{events: []event.Event{rejected, deadLettered, completed}}
	mux := mustTestDashboardMuxWithConfig(t, Config{Journal: journal, SeekableJournal: journal})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/dashboard/command-fates", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200\nbody:\n%s", response.Code, response.Body.String())
	}

	body := response.Body.String()

	for _, want := range []string{
		"Command Fates",
		"Newest 3 of 3 recorded fates",
		"rejected",
		"dead-lettered",
		"completed",
		"CreateUser",
		"SyncPush",
		"RenameTenant",
		"rejection/user.email_taken",
		"store unreachable",
		"/dashboard/events/",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body should contain %q\nbody:\n%s", want, body)
		}
	}
}

func TestCommandFatesIndexHandler_EmptyState(t *testing.T) {
	journal := &fakeSeekableJournal{events: []event.Event{makeTestEvent(t, "user.created", 1)}}
	mux := mustTestDashboardMuxWithConfig(t, Config{Journal: journal, SeekableJournal: journal})

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/dashboard/command-fates", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", response.Code)
	}

	body := response.Body.String()

	for _, want := range []string{
		"No command fates recorded",
		"ServiceConfig.CommandLifecycle",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("empty state should mention %q\nbody:\n%s", want, body)
		}
	}
}

func TestCommandFatesRoute_GatedByEventRead(t *testing.T) {
	// EventSource-only: a valid dashboard whose store answers stream-by-ref
	// reads but offers no journal — the exact state the route gate exists
	// for. (mustTestDashboardWithConfig cannot build this: it auto-wires a
	// journal whenever EventSource and Journal are both nil.)
	withoutJournal, err := New(Config{EventSource: memorystorage.NewMemoryStore()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, route := range withoutJournal.Routes() {
		if route.Pattern == "/command-fates" {
			t.Errorf("manifest should omit /command-fates without event-read capability, got %+v", route)
		}
	}

	journal := &fakeSeekableJournal{}
	withJournal := mustTestDashboardWithConfig(t, Config{Journal: journal, SeekableJournal: journal})

	found := false

	for _, route := range withJournal.Routes() {
		if route.Pattern == "/command-fates" && route.Method == http.MethodGet && !route.Write {
			found = true
		}
	}

	if !found {
		t.Error("manifest should carry GET /command-fates (read-only) when a journal is configured")
	}
}

func TestFateBadgeType(t *testing.T) {
	cases := map[string]display.BadgeType{
		"failed":        display.BadgeError,
		"dead-lettered": display.BadgeError,
		"rejected":      display.BadgeWarning,
		"retried":       display.BadgeWarning,
		"completed":     display.BadgeSuccess,
		"command.never-heard-of-it": display.BadgeNeutral,
	}

	for fate, want := range cases {
		if got := fateBadgeType(fate); got != want {
			t.Errorf("fateBadgeType(%q) = %v, want %v", fate, got, want)
		}
	}
}
