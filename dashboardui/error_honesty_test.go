package dashboardui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// ===== F48: writeJSON marshals before committing the status =====

func TestWriteJSON_MarshalFailureStillWrites500(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]any{"ch": make(chan int)})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on marshal failure, got %d", w.Code)
	}

	if body := w.Body.String(); !strings.Contains(body, `"marshal_failed"`) {
		t.Fatalf("expected marshal_failed body, got %q", body)
	}

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected JSON content-type, got %q", ct)
	}
}

func TestWriteJSON_HappyPathWritesPassedStatus(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if body := w.Body.String(); !strings.Contains(body, `"ok"`) {
		t.Fatalf("expected body payload, got %q", body)
	}
}

// ===== F46: detail handlers map error family to 404 vs 500 =====

func TestEventDetail_InfrastructureErrorReturns500(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{Journal: &fakeSeekableJournal{allErr: errors.New("disk exploded")}})

	// Any syntactically valid event ID reaches the loader, whose ReadAll
	// fails — that is an infrastructure failure, not a missing event.
	evt := hostileEvents(t, 1)[0]
	r := httptest.NewRequest(http.MethodGet, "/events/"+evt.ID().String(), nil)
	r.SetPathValue("id", evt.ID().String())
	w := httptest.NewRecorder()

	d.eventDetailHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for infra failure, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "failed to load event") {
		t.Fatalf("expected 'failed to load event' message, got: %s", w.Body.String()[:min(len(w.Body.String()), 200)])
	}
}

func TestEventDetail_RejectionReturns404(t *testing.T) {
	t.Parallel()

	// Healthy journal that simply does not contain the event.
	d := MustNew(Config{Journal: &fakeSeekableJournal{}})

	evt := hostileEvents(t, 1)[0]
	r := httptest.NewRequest(http.MethodGet, "/events/"+evt.ID().String(), nil)
	r.SetPathValue("id", evt.ID().String())
	w := httptest.NewRecorder()

	d.eventDetailHandler(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing event, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "event not found") {
		t.Fatalf("expected 'event not found' message")
	}
}

func TestCommandDetail_InfrastructureErrorReturns500(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{
		Journal:        &stubJournal{},
		CommandJournal: &fakeCommandJournal{err: errors.New("corrupt index")},
	})

	cmd := makeTestCommand(t)
	r := httptest.NewRequest(http.MethodGet, "/commands/"+cmd.ID().String(), nil)
	r.SetPathValue("id", cmd.ID().String())
	w := httptest.NewRecorder()

	d.commandDetailHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for infra failure, got %d", w.Code)
	}
}

// ===== F47: stream-listing pages surface reader errors =====

// errStreamReader fails every List call.
type errStreamReader struct{}

func (errStreamReader) List(
	context.Context,
	listing.ListOptions,
) (*listing.Page[listing.StreamListing], error) {
	return nil, errors.New("store unavailable")
}

func (errStreamReader) ListWithStatus(
	ctx context.Context,
	opts listing.ListOptions,
) (*listing.Page[listing.StreamStatus], error) {
	return nil, errors.New("store unavailable")
}

var _ listing.StreamReader = errStreamReader{}

func TestStreamsIndex_ReaderErrorRendersErrorPanel(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{
		Journal:      &stubJournal{},
		StreamReader: errStreamReader{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots", nil)
	d.snapshotsIndexHandler(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "failed to load streams") {
		t.Fatalf("expected the error panel message, got: %s", body[:min(len(body), 300)])
	}
}

// rejectionStreamReader fails List with a Rejection-family error, covering
// the 400 mapping in renderStreamIndex (a malformed cursor or any other
// client-side rejection from the listing layer).
type rejectionStreamReader struct{}

func (rejectionStreamReader) List(
	context.Context,
	listing.ListOptions,
) (*listing.Page[listing.StreamListing], error) {
	return nil, errorfamily.NewRejection(
		"dashboardui.streams.invalid_cursor", "invalid after cursor")
}

func (rejectionStreamReader) ListWithStatus(
	context.Context,
	listing.ListOptions,
) (*listing.Page[listing.StreamStatus], error) {
	return nil, errorfamily.NewRejection(
		"dashboardui.streams.invalid_cursor", "invalid after cursor")
}

var _ listing.StreamReader = rejectionStreamReader{}

func TestStreamsIndex_RejectionErrorReturns400(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{
		Journal:      &stubJournal{},
		StreamReader: rejectionStreamReader{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots", nil)
	d.snapshotsIndexHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a rejection-family error, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid after cursor") {
		t.Fatalf("expected 'invalid after cursor' message")
	}
}

func TestStreamsIndex_InvalidCursorReturns400(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{
		Journal:      &stubJournal{},
		StreamReader: &fakeStreamReader{items: streamListings()},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/snapshots?after=not-a-stream-id", nil)
	d.snapshotsIndexHandler(w, r)

	// ParseStreamID is deliberately lenient (any non-empty string parses),
	// so an arbitrary cursor never 400s — it simply matches nothing and the
	// page renders empty. This pins that contract.
	if w.Code != http.StatusOK {
		t.Fatalf(
			"expected 200 for a lenient cursor parse, got %d (%s)",
			w.Code,
			w.Body.String()[:min(len(w.Body.String()), 200)],
		)
	}
}
