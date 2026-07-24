package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
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

func TestDashboard_EventDetailRenders(t *testing.T) {
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

	url := "/dashboard/events/" + evt.ID().String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"user.created", "Payload", "Metadata", "Stream Type"} {
		if !strings.Contains(body, want) {
			t.Errorf("event detail page should contain %q", want)
		}
	}
}

func TestDashboard_AggregateDetailRenders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	aggID := id.NewAggregateID()
	ref := id.NewStreamRef("User", aggID)

	evt1, _ := event.New("user.created", aggID, "User", event.Version(1), struct{ Name string }{Name: "Alice"})
	_ = store.Save(nil, ref, []event.Event{evt1}, event.Version(0))

	evt2, _ := event.New("user.renamed", aggID, "User", event.Version(2), struct{ Name string }{Name: "Bob"})
	_ = store.Save(nil, ref, []event.Event{evt2}, event.Version(1))

	d, _ := New(Config{
		EventSource: store,
		Journal:     store,
	})

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	url := "/dashboard/aggregates/User/" + aggID.String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Event Timeline", "user.created", "user.renamed", "current version"} {
		if !strings.Contains(body, want) {
			t.Errorf("aggregate detail page should contain %q", want)
		}
	}
}

func TestDashboard_CommandAuditRenders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	cmdStore := memorystorage.NewMemoryCommandStore()

	aggID := id.NewAggregateID()
	ref := id.NewStreamRef("User", aggID)

	cmd, _ := command.NewPersistedCommand(
		"create.user",
		ref,
		[]byte(`{"name":"Alice"}`),
	)
	_ = cmdStore.Save(nil, ref, cmd)

	d, _ := New(Config{
		EventSource:    store,
		Journal:        store,
		CommandJournal: cmdStore,
	})

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/commands", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Command Audit", "create.user", "User"} {
		if !strings.Contains(body, want) {
			t.Errorf("commands page should contain %q", want)
		}
	}
}

func TestDashboard_QueryAuditRenders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	queryStore := memorystorage.NewMemoryQueryStore()

	q, _ := query.NewPersistedQuery("get.user", []byte(`{"id":"123"}`))
	_ = queryStore.SaveQuery(nil, q)

	d, _ := New(Config{
		EventSource:  store,
		Journal:      store,
		QueryJournal: queryStore,
	})

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/queries", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Query Audit", "get.user"} {
		if !strings.Contains(body, want) {
			t.Errorf("queries page should contain %q", want)
		}
	}
}

func TestDashboard_TimeTravelDetailRenders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	aggID := id.NewAggregateID()
	ref := id.NewStreamRef("Order", aggID)

	for i := 1; i <= 3; i++ {
		evt, _ := event.New(
			"order.updated",
			aggID,
			"Order",
			event.Version(i),
			struct{ Step int }{Step: i},
		)
		_ = store.Save(nil, ref, []event.Event{evt}, event.Version(i-1))
	}

	reader := listing.NewInMemoryStreamReader(store)

	d, _ := New(Config{
		EventSource:  store,
		Journal:      store,
		StreamReader: reader,
	})

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	// Test version 2 of 3
	url := "/dashboard/time-travel/Order/" + aggID.String() + "?v=2"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Time Travel", "Viewing version 2 of 3", "order.updated"} {
		if !strings.Contains(body, want) {
			t.Errorf("time-travel detail page should contain %q", want)
		}
	}
}

func TestDashboard_SnapshotDetailRenders(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	snapStore := memorystorage.NewMemorySnapshotStore()

	aggID := id.NewAggregateID()
	ref := id.NewStreamRef("Order", aggID)

	// Save a snapshot
	createdAt, _ := time.Parse(time.RFC3339, "2026-01-15T10:30:00Z")
	snap := snapshot.Snapshot{
		StreamID:   aggID,
		StreamType: "Order",
		Version:    event.Version(5),
		State:      []byte(`{"status":"shipped","total":42}`),
		CreatedAt:  createdAt,
	}
	_ = snapStore.Save(nil, snap)

	evt, _ := event.New("order.created", aggID, "Order", event.Version(1), struct{}{})
	_ = store.Save(nil, ref, []event.Event{evt}, event.Version(0))

	reader := listing.NewInMemoryStreamReader(store)

	d, _ := New(Config{
		EventSource:   store,
		Journal:       store,
		StreamReader:  reader,
		SnapshotStore: snapStore,
	})

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	url := "/dashboard/snapshots/Order/" + aggID.String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Snapshot", "Version", "shipped"} {
		if !strings.Contains(body, want) {
			t.Errorf("snapshot detail page should contain %q", want)
		}
	}
}

func TestDashboard_SSEBridgeWorks(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
		EventBus:    bus,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if d.broadcaster == nil {
		t.Fatal("broadcaster should be created when EventBus is configured")
	}

	if !d.Capabilities().EventBus {
		t.Fatal("EventBus capability should be detected")
	}

	// Subscribe to broadcaster
	ch := d.broadcaster.Subscribe()
	defer d.broadcaster.Unsubscribe(ch)

	// Publish an event to the bus
	aggID := id.NewAggregateID()

	evt, _ := event.New("order.placed", aggID, "Order", event.Version(1), struct{ Total int }{Total: 42})
	if err := bus.Publish(nil, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// The broadcaster should receive the event
	select {
	case sseEvt := <-ch:
		if sseEvt.ID.Get() != evt.ID().String() {
			t.Errorf("SSE event ID = %q, want %q", sseEvt.ID.Get(), evt.ID().String())
		}

	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE event from broadcaster")
	}
}
