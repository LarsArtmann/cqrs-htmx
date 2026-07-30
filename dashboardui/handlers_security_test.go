package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestXSS_EventTypeEscaped(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	aggID := id.NewStreamID()
	maliciousType := "<script>alert(1)</script>"
	evt, _ := event.New(event.Type(maliciousType), aggID, "User", event.Version(1), struct{}{})
	_ = store.Save(context.Background(), id.NewStreamRef("User", aggID), []event.Event{evt}, event.Version(0))

	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/events", nil)
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Errorf("XSS: raw <script> tag found in events page body")
	}

	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("expected escaped &lt;script&gt; in events page body")
	}
}

func TestXSS_EventDetailEscaped(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	aggID := id.NewStreamID()
	maliciousType := "<script>xss</script>"
	evt, _ := event.New(event.Type(maliciousType), aggID, "User", event.Version(1), struct{}{})
	_ = store.Save(context.Background(), id.NewStreamRef("User", aggID), []event.Event{evt}, event.Version(0))

	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/events/"+evt.ID().String(), nil)
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<script>xss</script>") {
		t.Errorf("XSS: raw <script> tag found in event detail body")
	}
}

func TestXSS_AggregateDetailEscaped(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	aggID := id.NewStreamID()
	maliciousType := "<script>agg</script>"
	evt, _ := event.New(event.Type(maliciousType), aggID, "User", event.Version(1), struct{}{})
	_ = store.Save(context.Background(), id.NewStreamRef("User", aggID), []event.Event{evt}, event.Version(0))

	reader := listing.NewInMemoryStreamReader(store)
	d, _ := New(Config{EventSource: store, Journal: store, StreamReader: reader})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/aggregates/User/"+aggID.String(), nil)
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "<script>agg</script>") {
		t.Errorf("XSS: raw <script> tag found in aggregate detail body")
	}
}

func TestOverviewStats_AccurateCount(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	for range 10 {
		aggID := id.NewStreamID()
		evt, _ := event.New("test.event", aggID, "TestAggregate", event.Version(1), struct{}{})
		_ = store.Save(
			context.Background(),
			id.NewStreamRef("TestAggregate", aggID),
			[]event.Event{evt},
			event.Version(0),
		)
	}

	reader := listing.NewInMemoryStreamReader(store)
	d, _ := New(Config{EventSource: store, SeekableJournal: store, StreamReader: reader})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, ">1<") && !strings.Contains(body, ">10<") && !strings.Contains(body, ">10<") {
		t.Errorf("overview stats should show accurate count for 10 events, not 1; check stat values")
	}
}

func TestNotFound_Handler(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/nonexistent-page", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Page Not Found") {
		t.Errorf("expected 'Page Not Found' in 404 body")
	}
}

func TestReadOnlyMode_WriteEndpointsNotFound(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{
		EventSource: store,
		Journal:     store,
		ReadOnly:    true,
	})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dashboard/projections/test/reset", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 404 or 405 for write endpoint in read-only mode, got %d", rec.Code)
	}
}
