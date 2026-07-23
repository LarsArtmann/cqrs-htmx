package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestDashboard_OverviewRenders(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Overview", "dashboard.css", "htmx.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("overview missing %q in body", want)
		}
	}
}

func TestDashboard_EventBrowserRenders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	aggID := id.NewAggregateID()
	evt, _ := event.New("user.created", aggID, "User", event.Version(1), struct{ Name string }{Name: "Alice"})
	_ = store.Save(nil, id.NewStreamRef("User", aggID), []event.Event{evt}, event.Version(0))

	d, _ := New(Config{
		EventSource: store,
		Journal:     store,
	})

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/events", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "user.created") {
		t.Errorf("events page should contain event type 'user.created'")
	}
}

func TestDashboard_CapabilityDetection(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	caps := d.Capabilities()
	if !caps.EventSource {
		t.Error("EventSource should be detected")
	}

	if !caps.Journal {
		t.Error("Journal should be detected")
	}

	if caps.SeekableJournal {
		t.Error("SeekableJournal should not be detected")
	}

	if caps.ProjectionHost {
		t.Error("ProjectionHost should not be detected")
	}
}

func TestDashboard_RequiresAtLeastOneInterface(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("New should return error when no interfaces provided")
	}
}

func TestDashboard_NavBuildsFromCapabilities(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{
		EventSource: store,
		Journal:     store,
	})

	p := d.page("Test", "/", &http.Request{})

	labels := make(map[string]bool)
	for _, n := range p.Nav {
		labels[n.Label] = true
	}

	if !labels["Overview"] {
		t.Error("nav should contain Overview")
	}

	if !labels["Events"] {
		t.Error("nav should contain Events")
	}

	if !labels["Aggregates"] {
		t.Error("nav should contain Aggregates")
	}

	if labels["Projections"] {
		t.Error("nav should NOT contain Projections (not configured)")
	}
}
