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

func TestEventFilter_ByType(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	aggID := id.NewStreamID()
	evt1, _ := event.New("user.created", aggID, "User", event.Version(1), struct{}{})
	_ = store.Save(context.Background(), id.NewStreamRef("User", aggID), []event.Event{evt1}, event.Version(0))

	aggID2 := id.NewStreamID()
	evt2, _ := event.New("user.deleted", aggID2, "User", event.Version(1), struct{}{})
	_ = store.Save(context.Background(), id.NewStreamRef("User", aggID2), []event.Event{evt2}, event.Version(0))

	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/events?type=user.created", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("filter status = %d; want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "user.created") {
		t.Errorf("filtered events should contain user.created")
	}

	if strings.Contains(body, "user.deleted") {
		t.Errorf("filtered events should NOT contain user.deleted")
	}
}

func TestCSS_ServedWithCorrectHeaders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard.css", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("css status = %d; want 200", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("css Content-Type = %q; want text/css", ct)
	}

	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age") {
		t.Errorf("css Cache-Control = %q; expected max-age", cc)
	}

	if !strings.Contains(rec.Body.String(), "--accent") {
		t.Errorf("css body should contain CSS custom property --accent")
	}
}

func TestJS_ServedWithCorrectHeaders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/dashboard.js", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("js status = %d; want 200", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Errorf("js Content-Type = %q; want text/javascript", ct)
	}

	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age") {
		t.Errorf("js Cache-Control = %q; expected max-age", cc)
	}
}

func TestPagination_PreservesFilterInLinks(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	for i := range 30 {
		aggID := id.NewStreamID()

		st := id.StreamType("User")
		if i >= 15 {
			st = "Order"
		}

		evt, _ := event.New(event.Type("entity.event"), aggID, st, event.Version(1), struct{}{})
		_ = store.Save(context.Background(), id.NewStreamRef(st, aggID), []event.Event{evt}, event.Version(0))
	}

	reader := listing.NewInMemoryStreamReader(store)
	d, _ := New(Config{EventSource: store, SeekableJournal: store, StreamReader: reader})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/events?streamType=User", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "Order") {
		t.Errorf("filtered by streamType=User should not contain Order events")
	}

	if strings.Contains(body, "pagination") {
		if !strings.Contains(body, "streamType=User") {
			t.Errorf("pagination links should preserve streamType=User filter")
		}
	}
}

func TestMiddleware_PermissionsPolicy(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})

	mw := d.Middleware()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	pp := rec.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Fatal("expected Permissions-Policy header to be set")
	}

	for _, want := range []string{"geolocation=()", "microphone=()", "camera=()", "payment=()", "usb=()"} {
		if !strings.Contains(pp, want) {
			t.Errorf("Permissions-Policy missing %q, got %q", want, pp)
		}
	}
}

func TestMiddleware_CSPWithNonceAndSecurityHeaders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})

	mw := d.Middleware()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected Content-Security-Policy header to be set")
	}

	if !strings.Contains(csp, "nonce-") {
		t.Errorf("expected CSP to contain a nonce (Nonce middleware should be in chain), got %q", csp)
	}

	if !strings.Contains(csp, "'self'") {
		t.Errorf("expected CSP to allow 'self', got %q", csp)
	}

	for _, tc := range []struct{ header, want string }{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	} {
		if got := rec.Header().Get(tc.header); got != tc.want {
			t.Errorf("%s = %q, want %q", tc.header, got, tc.want)
		}
	}
}
