package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// Hostile query values that must never inject additional query parameters
// into generated links: &, =, #, a space, and non-ASCII.
const hostileParamValue = "a&b=c#d e"

// hostileEvents seeds n events whose TYPE is the hostile value, so the
// filter matches and the table (with its sort headers and pagination links)
// actually renders.
func hostileEvents(t *testing.T, n int) []event.Event {
	t.Helper()

	streamID := id.NewStreamID()
	events := make([]event.Event, 0, n)

	for i := range n {
		evt, err := event.New(
			event.Type(hostileParamValue),
			streamID,
			"TestAgg",
			event.Version(i+1),
			map[string]string{"k": "v"},
		)
		if err != nil {
			t.Fatalf("event.New: %v", err)
		}

		events = append(events, evt)
	}

	return events
}

func TestSortState_ExtraParams_EscapesHostileColumn(t *testing.T) {
	t.Parallel()

	s := sortState{Column: "time&injected=1", Direction: sortAsc}
	got := s.extraParams()

	want := "sort=" + url.QueryEscape("time&injected=1") + "&dir=asc"
	if got != want {
		t.Fatalf("extraParams() = %q, want %q", got, want)
	}

	if vals, err := url.ParseQuery(got); err != nil {
		t.Fatalf("ParseQuery(%q): %v", got, err)
	} else if vals.Get("sort") != "time&injected=1" || vals.Get("dir") != "asc" {
		t.Fatalf("round-trip mismatch: %v", vals)
	}
}

func TestEventSortHeader_HrefEscapesColumn(t *testing.T) {
	t.Parallel()

	header := eventSortHeader("/d", "Time", "ty&pe", sortState{}, "")
	want := "/d/events?sort=" + url.QueryEscape("ty&pe") + "&dir=asc"

	if header.Href != want {
		t.Fatalf("href = %q, want %q", header.Href, want)
	}

	if strings.Contains(header.Href, "sort=ty&pe") {
		t.Fatalf("href carries unescaped column: %q", header.Href)
	}
}

// TestEventsIndex_HostileFilterRoundTripsEscaped pins the full loop: a
// hostile filter value arrives percent-encoded, matches events whose type IS
// that value, and every generated link (sort headers, pagination) carries it
// back escaped — never as raw &/= that would parse as extra parameters.
func TestEventsIndex_HostileFilterRoundTripsEscaped(t *testing.T) {
	t.Parallel()

	d := MustNew(Config{
		Journal: &fakeSeekableJournal{events: hostileEvents(t, 60)},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?type="+url.QueryEscape(hostileParamValue), nil)
	d.eventsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	escaped := url.QueryEscape(hostileParamValue)

	// Sort headers and the Next link all preserve the filter.
	if !strings.Contains(body, "type="+escaped) {
		t.Errorf("body should carry the escaped filter value %q", escaped)
	}

	// The raw value must never appear inside a link query string.
	if strings.Contains(body, "type="+hostileParamValue) {
		t.Errorf("body leaks unescaped filter value into links")
	}
}

// --- F43: malformed after cursors are a 400, not a silent page reset ---

func TestEventsIndex_InvalidAfterCursor_Returns400(t *testing.T) {
	t.Parallel()

	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?after=not-a-cursor", nil)
	d.eventsIndexHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid after cursor") {
		t.Fatalf("expected 'invalid after cursor' in body, got: %s", w.Body.String())
	}
}

func TestCommandsIndex_InvalidAfterCursor_Returns400(t *testing.T) {
	t.Parallel()

	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands?after=not-a-cursor", nil)
	d.commandsIndexHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestQueriesIndex_InvalidAfterCursor_Returns400(t *testing.T) {
	t.Parallel()

	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/queries?after=not-a-cursor", nil)
	d.queriesIndexHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- F44: the ReadAll fallback paginates with the parsed page size ---

type readAllCommandJournal struct{ cmds []*command.PersistedCommand }

func (j *readAllCommandJournal) ReadAll(context.Context) ([]*command.PersistedCommand, error) {
	return j.cmds, nil
}

var _ command.CommandJournal = (*readAllCommandJournal)(nil)

func TestCommandsIndex_FallbackHonorsParsedPageSize(t *testing.T) {
	t.Parallel()

	cmds := make([]*command.PersistedCommand, 0, 30)
	for range 30 {
		cmds = append(cmds, makeTestCommand(t))
	}

	// Journal is NOT seekable, so the ReadAll fallback path runs.
	d := MustNew(Config{CommandJournal: &readAllCommandJournal{cmds: cmds}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands?limit=10", nil)
	d.commandsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	// 30 commands at limit=10: the fallback must report a next page (the
	// old early-truncation computed hasNext on the already-truncated slice,
	// so the Next link never appeared) and truncate with the PARSED page
	// size, not the config default.
	if !strings.Contains(body, "Next") {
		t.Error("expected a Next link for 30 commands at limit=10")
	}

	if !strings.Contains(body, "after=") {
		t.Error("expected the Next link to carry an after cursor")
	}

	if !strings.Contains(body, "Showing 1–10") {
		t.Errorf("expected 'Showing 1–10' for limit=10, got body prefix: %s", body[:min(len(body), 400)])
	}
}

type readAllQueryJournal struct{ queries []*query.PersistedQuery }

func (j *readAllQueryJournal) ReadAllQueries(context.Context) ([]*query.PersistedQuery, error) {
	return j.queries, nil
}

var _ query.QueryJournal = (*readAllQueryJournal)(nil)

func TestQueriesIndex_FallbackHonorsParsedPageSize(t *testing.T) {
	t.Parallel()

	queries := make([]*query.PersistedQuery, 0, 30)
	for range 30 {
		queries = append(queries, makeTestQuery(t))
	}

	d := MustNew(Config{QueryJournal: &readAllQueryJournal{queries: queries}})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/queries?limit=10", nil)
	d.queriesIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, "Next") {
		t.Error("expected a Next link for 30 queries at limit=10")
	}

	// The old fallback truncated with d.config.PageSize (50), showing all
	// 30 rows while claiming one page.
	if !strings.Contains(body, "Showing 1–10") {
		t.Error("expected 'Showing 1–10' for limit=10")
	}
}
