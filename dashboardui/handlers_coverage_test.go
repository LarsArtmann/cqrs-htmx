package dashboardui

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ===== Aggregates Index Handler =====

func TestAggregatesIndexHandler_NoStreamReader(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/aggregates", nil)
	d.aggregatesIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "No aggregates found") {
		t.Fatalf("expected empty state, got: %s", w.Body.String())
	}
}

func TestAggregatesIndexHandler_WithListings(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:      &stubJournal{},
		StreamReader: &fakeStreamReader{items: streamListings()},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/aggregates", nil)
	d.aggregatesIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	for _, want := range []string{"Aggregates", "user", "tenant", "Last Event"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in body", want)
		}
	}
}

func TestAggregatesIndexHandler_PaginationHasMore(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:      &stubJournal{},
		StreamReader: &fakeStreamReader{items: streamListings()},
		PageSize:     1,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/aggregates", nil)
	d.aggregatesIndexHandler(w, r)

	body := w.Body.String()

	if !strings.Contains(body, "Next") {
		t.Fatalf("expected pagination Next link when HasMore, got: %s", body)
	}

	if !strings.Contains(body, "after=") {
		t.Fatalf("expected after= cursor in pagination link, got: %s", body)
	}
}

func TestAggregatesIndexHandler_PaginationHasPrev(t *testing.T) {
	listings := streamListings()
	cursor := listings[0].ID.String()

	d := mustTestDashboardWithConfig(t, Config{
		Journal:      &stubJournal{},
		StreamReader: &fakeStreamReader{items: listings},
		PageSize:     1,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/aggregates?after="+cursor, nil)
	d.aggregatesIndexHandler(w, r)

	body := w.Body.String()

	if !strings.Contains(body, "Previous") {
		t.Fatalf("expected pagination Previous link when after param present, got: %s", body)
	}
}

// ===== Projection Health Partial Handler =====

func TestProjectionHealthPartialHandler_NilHost(t *testing.T) {
	d := mustTestDashboard(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/-/partials/projection-health", nil)
	d.projectionHealthPartialHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "projection-health") {
		t.Fatalf("expected projection-health panel div, got: %s", body)
	}

	if !strings.Contains(body, "Projection Health") {
		t.Fatalf("expected panel title, got: %s", body)
	}
}

func TestProjectionHealthPartialHandler_EmptyHost(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:        &stubJournal{},
		ProjectionHost: &projectionhost.Host{},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/-/partials/projection-health", nil)
	d.projectionHealthPartialHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Projection Health") {
		t.Fatalf("expected panel title")
	}
}

// ===== Projection Status Classification =====

func TestProjectionStatusKind(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{"running", statusGood},
		{"live", statusGood},
		{"RUNNING", statusGood},
		{"idle", statusWarn},
		{"backoff", statusWarn},
		{"draining", statusWarn},
		{"stopped", statusBad},
		{"failed", statusBad},
		{"unknown", statusNeutral},
		{"", statusNeutral},
	}

	for _, tc := range cases {
		got := projectionStatusKind(tc.status)
		if got != tc.want {
			t.Errorf("projectionStatusKind(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// ===== Projection Row Rendering =====

func TestRenderProjectionRow_AllStatusKinds(t *testing.T) {
	cases := []struct {
		name       string
		statusKind string
		wantClass  string
	}{
		{"good-proj", statusGood, "badge badge-ok"},
		{"warn-proj", statusWarn, "badge badge-warn"},
		{"bad-proj", statusBad, "badge badge-err"},
		{"neutral-proj", statusNeutral, "badge badge-neutral"},
	}

	for _, tc := range cases {
		html := renderProjectionRow(projectionStat{ //nolint:exhaustruct // test fixture
			Name: tc.name, Status: "active", StatusKind: tc.statusKind,
			Processed: 100, Errors: 2,
		})
		if !strings.Contains(html, tc.wantClass) {
			t.Errorf("renderProjectionRow(%s): expected class %q, got: %s", tc.name, tc.wantClass, html)
		}

		if !strings.Contains(html, tc.name) {
			t.Errorf("renderProjectionRow(%s): expected name in output", tc.name)
		}
	}
}

// ===== Projection Health Panel Rendering =====

func TestRenderProjectionHealthPanel_Empty(t *testing.T) {
	html := renderProjectionHealthPanel("/dashboard", nil)

	if !strings.Contains(html, "projection-health") {
		t.Fatalf("expected panel div id")
	}

	if !strings.Contains(html, "Projection Health") {
		t.Fatalf("expected panel title")
	}

	if strings.Contains(html, "<tr><td>") {
		t.Fatalf("expected no projection rows for empty list")
	}
}

func TestRenderProjectionHealthPanel_WithProjections(t *testing.T) {
	projs := []projectionStat{
		{ //nolint:exhaustruct // test fixture
			Name: "user-read-model", Status: "running", StatusKind: statusGood,
			Lag: "0s", Processed: 500, Errors: 0,
		},
		{ //nolint:exhaustruct // test fixture
			Name: "casbin-projection", Status: "failed", StatusKind: statusBad,
			Lag: "5m", Processed: 300, Errors: 3,
		},
	}

	html := renderProjectionHealthPanel("/dashboard", projs)

	for _, want := range []string{"user-read-model", "casbin-projection", "badge-ok", "badge-err", "500", "3"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in panel HTML", want)
		}
	}

	if !strings.Contains(html, `hx-trigger="every 10s"`) {
		t.Errorf("expected HTMX polling attribute")
	}
}

// ===== buildProjectionStats =====

func TestBuildProjectionStats_NilHost(t *testing.T) {
	if stats := buildProjectionStats(nil); stats != nil {
		t.Fatalf("expected nil for nil host, got %v", stats)
	}
}

func TestBuildProjectionStats_EmptyHost(t *testing.T) {
	host := &projectionhost.Host{}

	stats := buildProjectionStats(host)
	if len(stats) != 0 {
		t.Fatalf("expected empty stats for zero-value host, got %d entries", len(stats))
	}
}

// ===== MustNew =====

func TestMustNew_Success(t *testing.T) {
	d := MustNew(Config{Journal: &stubJournal{}})
	if d == nil {
		t.Fatal("expected non-nil Dashboard")
	}

	if d.cfg.PageSize != defaultPageSize {
		t.Errorf("expected default PageSize %d, got %d", defaultPageSize, d.cfg.PageSize)
	}
}

func TestMustNew_PanicsOnInvalidConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid config")
		}
	}()

	MustNew(Config{})
}

// ===== Guard =====

func TestGuard_NilAuthorizer(t *testing.T) {
	d := mustTestDashboard(t)

	called := false
	h := d.guard(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)

	if !called {
		t.Fatal("expected handler called when authorizer is nil")
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGuard_Denied(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:    &stubJournal{},
		Authorizer: func(*http.Request) error { return errors.New("denied") },
	})

	called := false
	h := d.guard(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)

	if called {
		t.Fatal("expected handler NOT called when authorizer denies")
	}

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "forbidden") {
		t.Fatalf("expected 'forbidden' in body, got: %s", w.Body.String())
	}
}

func TestGuard_Allowed(t *testing.T) {
	d := mustTestDashboardWithConfig(t, Config{
		Journal:    &stubJournal{},
		Authorizer: func(*http.Request) error { return nil },
	})

	called := false
	h := d.guard(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)

	if !called {
		t.Fatal("expected handler called when authorizer allows")
	}
}

// ===== Accessors =====

func TestHandler_ReturnsNonNil(t *testing.T) {
	d := mustTestDashboard(t)
	if d.Handler() == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestMiddleware_ReturnsNonNil(t *testing.T) {
	d := mustTestDashboard(t)

	mw := d.Middleware()
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}

	called := false
	wrapped := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(w, r)

	if !called {
		t.Fatal("expected inner handler called through middleware")
	}
}

func TestConfig_ReturnsWithDefaults(t *testing.T) {
	d := mustTestDashboard(t)
	cfg := d.Config()

	if cfg.PageSize != defaultPageSize {
		t.Errorf("expected default PageSize, got %d", cfg.PageSize)
	}

	if cfg.Title == "" {
		t.Error("expected non-empty Title after defaults")
	}

	if cfg.BasePath == "" {
		t.Error("expected non-empty BasePath after defaults")
	}
}

// ===== relativeTime =====

func TestRelativeTime(t *testing.T) {
	if got := relativeTime(time.Time{}); got != "" {
		t.Errorf("relativeTime(zero) = %q, want empty", got)
	}

	if got := relativeTime(time.Now()); got != "just now" {
		t.Errorf("relativeTime(now) = %q, want %q", got, "just now")
	}

	if got := relativeTime(time.Now().Add(-90 * time.Second)); got != "1 minute ago" {
		t.Errorf("relativeTime(-90s) = %q, want %q", got, "1 minute ago")
	}

	if got := relativeTime(time.Now().Add(-5 * time.Minute)); !strings.Contains(got, "minutes ago") {
		t.Errorf("relativeTime(-5m) = %q, want 'minutes ago'", got)
	}

	if got := relativeTime(time.Now().Add(-90 * time.Minute)); got != "1 hour ago" {
		t.Errorf("relativeTime(-90m) = %q, want %q", got, "1 hour ago")
	}

	if got := relativeTime(time.Now().Add(-3 * time.Hour)); !strings.Contains(got, "hours ago") {
		t.Errorf("relativeTime(-3h) = %q, want 'hours ago'", got)
	}

	if got := relativeTime(time.Now().AddDate(0, 0, -1)); got != "1 day ago" {
		t.Errorf("relativeTime(-1d) = %q, want %q", got, "1 day ago")
	}

	if got := relativeTime(time.Now().AddDate(0, 0, -5)); !strings.Contains(got, "days ago") {
		t.Errorf("relativeTime(-5d) = %q, want 'days ago'", got)
	}

	if got := relativeTime(time.Now().AddDate(0, -1, 0)); !strings.Contains(got, "month") {
		t.Errorf("relativeTime(-1mo) = %q, want contains 'month'", got)
	}

	if got := relativeTime(time.Now().AddDate(0, -3, 0)); !strings.Contains(got, "months ago") {
		t.Errorf("relativeTime(-3mo) = %q, want 'months ago'", got)
	}

	if got := relativeTime(time.Now().AddDate(-2, 0, 0)); !strings.Contains(got, "years ago") {
		t.Errorf("relativeTime(-2y) = %q, want 'years ago'", got)
	}
}

// ===== humanByteSize =====

func TestHumanByteSize(t *testing.T) {
	cases := []struct {
		bytes int
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}

	for _, tc := range cases {
		got := humanByteSize(tc.bytes)
		if got != tc.want {
			t.Errorf("humanByteSize(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

// ===== encodingBadgeClass =====

func TestEncodingBadgeClass(t *testing.T) {
	cases := []struct {
		encoding string
		want     string
	}{
		{"json", "badge badge-neutral"},
		{"", "badge badge-neutral"},
		{"cbor", "badge badge-warn"},
		{"raw", "badge badge-neutral"},
		{"unknown-format", "badge badge-neutral"},
	}

	for _, tc := range cases {
		got := encodingBadgeClass(tc.encoding)
		if got != tc.want {
			t.Errorf("encodingBadgeClass(%q) = %q, want %q", tc.encoding, got, tc.want)
		}
	}
}

// ===== parsePageSize =====

func TestParsePageSize(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		defaultVal int
		want       int
	}{
		{"no limit param", "", 50, 50},
		{"valid limit", "limit=10", 50, 10},
		{"invalid limit", "limit=abc", 50, 50},
		{"zero limit", "limit=0", 50, 50},
		{"negative limit", "limit=-5", 50, 50},
		{"over max", "limit=500", 50, maxPageSize},
		{"exactly max", "limit=200", 50, maxPageSize},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/?"
			if tc.query != "" {
				url += tc.query
			}

			r := httptest.NewRequest(http.MethodGet, url, nil)

			got := parsePageSize(r, tc.defaultVal)
			if got != tc.want {
				t.Errorf("parsePageSize(%q) = %d, want %d", tc.query, got, tc.want)
			}
		})
	}
}

// ===== renderPagination =====

func TestRenderPagination_NoPagination(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{}, "") //nolint:exhaustruct // zero-value
	if html != "" {
		t.Fatalf("expected empty string for no pagination, got: %s", html)
	}
}

func TestRenderPagination_HasNextOnly(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{ //nolint:exhaustruct // test fixture
		HasNext: true, NextCursor: "abc", PageSize: 10,
	}, "")

	if !strings.Contains(html, "Next") {
		t.Errorf("expected Next link")
	}

	if strings.Contains(html, "Previous</a>") {
		t.Errorf("did not expect Previous as a link (should be disabled span)")
	}

	if !strings.Contains(html, "after=abc") {
		t.Errorf("expected cursor in link")
	}
}

func TestRenderPagination_HasPrevOnly(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{ //nolint:exhaustruct // test fixture
		HasPrev: true,
	}, "")

	if !strings.Contains(html, "Previous") {
		t.Errorf("expected Previous link")
	}

	if strings.Contains(html, "after=") {
		t.Errorf("did not expect Next link/cursor")
	}
}

func TestRenderPagination_Both(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{ //nolint:exhaustruct // test fixture
		HasNext: true, NextCursor: "xyz", PageSize: 20, HasPrev: true,
	}, "")

	if !strings.Contains(html, "Previous") {
		t.Errorf("expected Previous link")
	}

	if !strings.Contains(html, "Next") {
		t.Errorf("expected Next link")
	}
}

func TestRenderPagination_WithExtraParams(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{ //nolint:exhaustruct // test fixture
		HasNext: true, NextCursor: "cur", PageSize: 10,
	}, "type=user.created")

	if !strings.Contains(html, "type=user.created") {
		t.Errorf("expected extra params preserved in pagination link")
	}
}

// ===== List Streams Paged Helper =====

func TestListStreamsPaged_NilReader(t *testing.T) {
	d := mustTestDashboard(t)
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	streams, _ := d.listStreamsPaged(r)
	if streams != nil {
		t.Fatalf("expected nil for nil StreamReader, got %v", streams)
	}
}

func TestListStreamsPaged_WithReader(t *testing.T) {
	expected := streamListings()
	d := mustTestDashboardWithConfig(t, Config{
		Journal:      &stubJournal{},
		StreamReader: &fakeStreamReader{items: expected},
	})
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	streams, page := d.listStreamsPaged(r)
	if len(streams) != len(expected) {
		t.Fatalf("expected %d streams, got %d", len(expected), len(streams))
	}

	if page.PageSize != defaultPageSize {
		t.Errorf("expected page size %d, got %d", defaultPageSize, page.PageSize)
	}
}

// ===== Meta Row Helpers =====

func TestMetaRow(t *testing.T) {
	var b strings.Builder
	metaRow(&b, "Type", "user.created")

	if !strings.Contains(b.String(), "Type") || !strings.Contains(b.String(), "user.created") {
		t.Fatalf("expected key and value in meta row, got: %s", b.String())
	}
}

func TestMetaRowCopyable(t *testing.T) {
	var b strings.Builder
	metaRowCopyable(&b, "ID", "display-id", "raw-id-val")

	html := b.String()
	if !strings.Contains(html, "data-copyable=\"raw-id-val\"") {
		t.Errorf("expected data-copyable attribute with raw value")
	}

	if !strings.Contains(html, "display-id") {
		t.Errorf("expected display value in output")
	}
}

// ===== Truncate + Esc =====

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate(short, 10) = %q, want %q", got, "short")
	}

	if got := truncate("a-very-long-id-string", 5); got != "a-ver..." {
		t.Errorf("truncate(long, 5) = %q, want %q", got, "a-ver...")
	}
}

func TestEsc(t *testing.T) {
	if got := esc("<script>"); got != "&lt;script&gt;" {
		t.Errorf("esc(<script>) = %q, want %q", got, "&lt;script&gt;")
	}
}

// ===== Empty State =====

func TestEmptyState_NoMessage(t *testing.T) {
	html := emptyState("Nothing Here", "")
	if !strings.Contains(html, "Nothing Here") {
		t.Errorf("expected title in output")
	}

	if strings.Contains(html, "<p>") {
		t.Errorf("did not expect <p> tag when message is empty")
	}
}

func TestEmptyState_WithMessage(t *testing.T) {
	html := emptyState("Nothing Here", "Try again later")
	if !strings.Contains(html, "Try again later") {
		t.Errorf("expected message in output")
	}
}
