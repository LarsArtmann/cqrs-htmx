package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

func makeStreamListings(n int) []listing.StreamListing {
	items := make([]listing.StreamListing, n)
	for i := range items {
		items[i] = listing.StreamListing{
			ID:         id.NewStreamID(),
			Type:       id.StreamType("User"),
			Version:    event.Version(i + 1),
			EventCount: uint(i + 1),
		}
	}

	return items
}

func TestListStreamsPaged_NilReader(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/?", nil)

	listings, state := ListStreamsPaged(r, Config{PageSize: 20})
	if listings != nil {
		t.Errorf("expected nil listings, got %d", len(listings))
	}

	if state.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", state.PageSize)
	}

	if state.HasNext {
		t.Error("HasNext should be false for nil reader")
	}
}

func TestListStreamsPaged_EmptyResult(t *testing.T) {
	t.Parallel()

	cfg := Config{
		PageSize:     20,
		StreamReader: &fakeStreamReader{page: &listing.Page[listing.StreamListing]{}},
	}
	r := httptest.NewRequest(http.MethodGet, "/?", nil)

	listings, state := ListStreamsPaged(r, cfg)
	if len(listings) != 0 {
		t.Errorf("expected 0 listings, got %d", len(listings))
	}

	if state.HasNext {
		t.Error("HasNext should be false for empty result")
	}
}

func TestListStreamsPaged_SinglePage(t *testing.T) {
	t.Parallel()

	items := makeStreamListings(5)
	cfg := Config{
		PageSize: 20,
		StreamReader: &fakeStreamReader{
			page: &listing.Page[listing.StreamListing]{Items: items},
		},
	}
	r := httptest.NewRequest(http.MethodGet, "/?", nil)

	listings, state := ListStreamsPaged(r, cfg)
	if len(listings) != 5 {
		t.Fatalf("expected 5 listings, got %d", len(listings))
	}

	if state.HasNext {
		t.Error("HasNext should be false when items ≤ pageSize")
	}

	if state.NextCursor != "" {
		t.Errorf("NextCursor should be empty, got %q", state.NextCursor)
	}
}

func TestListStreamsPaged_MultiPage(t *testing.T) {
	t.Parallel()

	pageSize := 10
	items := makeStreamListings(pageSize + 5)
	cfg := Config{
		PageSize: pageSize,
		StreamReader: &fakeStreamReader{
			page: &listing.Page[listing.StreamListing]{Items: items},
		},
	}
	r := httptest.NewRequest(http.MethodGet, "/?", nil)

	listings, state := ListStreamsPaged(r, cfg)
	if len(listings) != pageSize {
		t.Fatalf("expected %d listings (trimmed to pageSize), got %d", pageSize, len(listings))
	}

	if !state.HasNext {
		t.Error("HasNext should be true when more items exist")
	}

	if state.NextCursor == "" {
		t.Error("NextCursor should be set when HasNext")
	}

	if state.NextCursor != listings[len(listings)-1].ID.String() {
		t.Errorf("NextCursor = %q, want last item ID %q",
			state.NextCursor, listings[len(listings)-1].ID.String())
	}
}

func TestListStreamsPaged_WithCursor(t *testing.T) {
	t.Parallel()

	afterID := id.NewStreamID()
	items := makeStreamListings(3)
	cfg := Config{
		PageSize: 20,
		StreamReader: &fakeStreamReader{
			page: &listing.Page[listing.StreamListing]{Items: items},
		},
	}
	r := httptest.NewRequest(http.MethodGet, "/?after="+afterID.String(), nil)

	_, state := ListStreamsPaged(r, cfg)
	if !state.HasPrev {
		t.Error("HasPrev should be true when after cursor is present")
	}

	if state.After != afterID.String() {
		t.Errorf("After = %q, want %q", state.After, afterID.String())
	}
}

func TestListStreamsPaged_ErrorReturnsNil(t *testing.T) {
	t.Parallel()

	cfg := Config{
		PageSize:     20,
		StreamReader: &fakeStreamReader{err: context.Canceled},
	}
	r := httptest.NewRequest(http.MethodGet, "/?", nil)

	listings, state := ListStreamsPaged(r, cfg)
	if listings != nil {
		t.Errorf("expected nil listings on error, got %d", len(listings))
	}

	if state.HasNext {
		t.Error("HasNext should be false on error")
	}
}

func TestListStreamsPaged_NilPageReturnsNil(t *testing.T) {
	t.Parallel()

	cfg := Config{
		PageSize:     20,
		StreamReader: &fakeStreamReader{page: nil},
	}
	r := httptest.NewRequest(http.MethodGet, "/?", nil)

	listings, _ := ListStreamsPaged(r, cfg)
	if listings != nil {
		t.Errorf("expected nil listings for nil page, got %d", len(listings))
	}
}

func TestFetchOverview_WithStreamReaderData(t *testing.T) {
	t.Parallel()

	items := makeStreamListings(3)
	cfg := Config{
		PageSize: 50,
		StreamReader: &fakeStreamReader{
			page: &listing.Page[listing.StreamListing]{Items: items},
		},
	}

	overview := FetchOverview(context.Background(), cfg)
	if overview.TotalAggregates != "3" {
		t.Errorf("TotalAggregates = %q, want %q", overview.TotalAggregates, "3")
	}
}

func TestFetchOverview_StreamReaderWithMore(t *testing.T) {
	t.Parallel()

	items := makeStreamListings(50)
	cfg := Config{
		PageSize: 50,
		StreamReader: &fakeStreamReader{
			page: &listing.Page[listing.StreamListing]{Items: items, HasMore: true},
		},
	}

	overview := FetchOverview(context.Background(), cfg)
	if overview.TotalAggregates != "50+" {
		t.Errorf("TotalAggregates = %q, want %q", overview.TotalAggregates, "50+")
	}
}

func TestFetchOverview_StreamReaderError(t *testing.T) {
	t.Parallel()

	cfg := Config{
		PageSize:     50,
		StreamReader: &fakeStreamReader{err: context.Canceled},
	}

	overview := FetchOverview(context.Background(), cfg)
	if overview.TotalAggregates != "0" {
		t.Errorf("TotalAggregates = %q, want %q on error", overview.TotalAggregates, "0")
	}
}

func TestFetchOverview_SeekableJournalReadError(t *testing.T) {
	t.Parallel()

	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{readErr: context.Canceled},
	}

	overview := FetchOverview(context.Background(), cfg)
	if overview.TotalEvents != "0" {
		t.Errorf("TotalEvents = %q, want %q on error", overview.TotalEvents, "0")
	}

	if len(overview.RecentEvents) != 0 {
		t.Errorf("RecentEvents should be empty on error, got %d", len(overview.RecentEvents))
	}
}

func TestFetchOverview_JournalReadError(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Journal: &fakeSeekableJournal{allErr: context.Canceled},
	}

	overview := FetchOverview(context.Background(), cfg)
	if overview.TotalEvents != "0" {
		t.Errorf("TotalEvents = %q, want %q on error", overview.TotalEvents, "0")
	}
}

func TestFetchOverview_SeekableJournalAtCountLimit(t *testing.T) {
	t.Parallel()

	events := make([]event.Event, OverviewCountLimit)
	for i := range events {
		events[i] = makeTestEvent("test", i+1)
	}

	cfg := Config{
		SeekableJournal: &fakeSeekableJournal{events: events},
	}

	overview := FetchOverview(context.Background(), cfg)
	if overview.TotalEvents != "500+" {
		t.Errorf("TotalEvents = %q, want %q at count limit", overview.TotalEvents, "500+")
	}
}

func TestProjectionStats_WithHost(t *testing.T) {
	t.Parallel()

	journal := &fakeSeekableJournal{}

	host, err := projectionhost.New(journal, fakeCheckpointStore{})
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(testProjection{name: "user-read-model"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	stats := ProjectionStats(host)
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}

	if stats[0].Name != "user-read-model" {
		t.Errorf("Name = %q, want %q", stats[0].Name, "user-read-model")
	}
}

func TestDLQProjectionLinks_WithHostFallback(t *testing.T) {
	t.Parallel()

	journal := &fakeSeekableJournal{}

	host, err := projectionhost.New(journal, fakeCheckpointStore{})
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(testProjection{name: "casbin-projection"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	cfg := Config{ProjectionHost: host}

	links := DLQProjectionLinks(context.Background(), cfg)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}

	if links[0].Name != "casbin-projection" {
		t.Errorf("Name = %q, want %q", links[0].Name, "casbin-projection")
	}
	// No DeadLetterStore → fallback to ws.Errors (0 for fresh host)
	if links[0].Count != 0 {
		t.Errorf("Count = %d, want 0 for fresh host", links[0].Count)
	}
}

func TestDLQProjectionLinks_WithDeadLetterStore(t *testing.T) {
	t.Parallel()

	journal := &fakeSeekableJournal{}

	host, err := projectionhost.New(journal, fakeCheckpointStore{})
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(testProjection{name: "user-read-model"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	cfg := Config{
		ProjectionHost:  host,
		DeadLetterStore: &fakeDeadLetterStore{entries: make([]projectionhost.DeadLetterEntry, 3)},
	}

	links := DLQProjectionLinks(context.Background(), cfg)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}

	if links[0].Count != 3 {
		t.Errorf("Count = %d, want 3 from DeadLetterStore", links[0].Count)
	}
}

func TestFetchOverview_WithProjectionHost(t *testing.T) {
	t.Parallel()

	journal := &fakeSeekableJournal{}

	host, err := projectionhost.New(journal, fakeCheckpointStore{})
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(testProjection{name: "user-read-model"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	cfg := Config{ProjectionHost: host}

	overview := FetchOverview(context.Background(), cfg)
	if len(overview.Projections) != 1 {
		t.Fatalf("expected 1 projection, got %d", len(overview.Projections))
	}
	// Fresh host projections are idle (not started) → Degraded/warn.
	if overview.HealthStatus != "Degraded" {
		t.Errorf("HealthStatus = %q, want %q", overview.HealthStatus, "Degraded")
	}

	if overview.HealthKind != StatusWarn {
		t.Errorf("HealthKind = %q, want %q", overview.HealthKind, StatusWarn)
	}
}
