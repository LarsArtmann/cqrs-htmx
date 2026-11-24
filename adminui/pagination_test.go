package adminui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestPageBounds(t *testing.T) {
	tests := []struct {
		name                        string
		page, total, pageSize       int
		wantOffset, wantLimit       int
		wantTotalPages, wantCurrent int
	}{
		{
			name:           "empty list is one empty page",
			page:           1,
			total:          0,
			pageSize:       50,
			wantOffset:     0,
			wantLimit:      0,
			wantTotalPages: 1,
			wantCurrent:    1,
		},
		{
			name:           "single partial page",
			page:           1,
			total:          7,
			pageSize:       50,
			wantOffset:     0,
			wantLimit:      7,
			wantTotalPages: 1,
			wantCurrent:    1,
		},
		{
			name:           "exact fit",
			page:           1,
			total:          100,
			pageSize:       50,
			wantOffset:     0,
			wantLimit:      50,
			wantTotalPages: 2,
			wantCurrent:    1,
		},
		{
			name:           "second page",
			page:           2,
			total:          120,
			pageSize:       50,
			wantOffset:     50,
			wantLimit:      50,
			wantTotalPages: 3,
			wantCurrent:    2,
		},
		{
			name:           "last partial page",
			page:           3,
			total:          120,
			pageSize:       50,
			wantOffset:     100,
			wantLimit:      20,
			wantTotalPages: 3,
			wantCurrent:    3,
		},
		{
			name:           "page zero clamps to first",
			page:           0,
			total:          120,
			pageSize:       50,
			wantOffset:     0,
			wantLimit:      50,
			wantTotalPages: 3,
			wantCurrent:    1,
		},
		{
			name:           "negative page clamps to first",
			page:           -4,
			total:          120,
			pageSize:       50,
			wantOffset:     0,
			wantLimit:      50,
			wantTotalPages: 3,
			wantCurrent:    1,
		},
		{
			name:           "beyond range clamps to last",
			page:           99,
			total:          120,
			pageSize:       50,
			wantOffset:     100,
			wantLimit:      20,
			wantTotalPages: 3,
			wantCurrent:    3,
		},
		{
			name:           "pageSize zero falls back to DefaultPageSize",
			page:           1,
			total:          80,
			pageSize:       0,
			wantOffset:     0,
			wantLimit:      50,
			wantTotalPages: 2,
			wantCurrent:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, limit, totalPages, current := pageBounds(tt.page, tt.total, tt.pageSize)
			if offset != tt.wantOffset || limit != tt.wantLimit || totalPages != tt.wantTotalPages ||
				current != tt.wantCurrent {
				t.Errorf("pageBounds(%d, %d, %d) = (%d, %d, %d, %d), want (%d, %d, %d, %d)",
					tt.page, tt.total, tt.pageSize,
					offset, limit, totalPages, current,
					tt.wantOffset, tt.wantLimit, tt.wantTotalPages, tt.wantCurrent)
			}
			if offset+limit > tt.total && tt.total > 0 {
				t.Errorf("window [%d,%d) exceeds total %d", offset, offset+limit, tt.total)
			}
		})
	}
}

func TestParsePageQuery(t *testing.T) {
	tests := []struct {
		raw  string
		want int
	}{
		{raw: "", want: 1},
		{raw: "1", want: 1},
		{raw: "3", want: 3},
		{raw: "0", want: 1},
		{raw: "-2", want: 1},
		{raw: "abc", want: 1},
		{raw: "12x", want: 1},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/admin/users?page="+tt.raw, nil)
		if got := parsePageQuery(req); got != tt.want {
			t.Errorf("parsePageQuery(page=%q) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}

func TestListPageURL(t *testing.T) {
	if got := listPageURL("/admin/users", ""); got != "/admin/users" {
		t.Errorf("no search: got %q, want /admin/users", got)
	}
	if got := listPageURL("/admin/users", "alice x"); got != "/admin/users?q=alice+x" {
		t.Errorf("search: got %q, want /admin/users?q=alice+x", got)
	}
}

// TestPagerWindow pins the pager's windowing: three pages around the current
// one, first and last always present, ellipsis markers where numbers were
// dropped, out-of-range pages clamped, and tiny lists rendered in full.
func TestPagerWindow(t *testing.T) {
	tests := []struct {
		name        string
		page, total int
		want        []pagerStep
	}{
		{"no pages at all", 1, 0, nil},
		{"negative total", 1, -3, nil},
		{"single page", 1, 1, []pagerStep{{Page: 1, Ellipsis: false}}},
		{"three pages render in full", 2, 3, []pagerStep{
			{Page: 1, Ellipsis: false}, {Page: 2, Ellipsis: false}, {Page: 3, Ellipsis: false},
		}},
		{"first page of many", 1, 10, []pagerStep{
			{Page: 1, Ellipsis: false}, {Page: 2, Ellipsis: false}, {Page: 3, Ellipsis: false},
			{Page: 0, Ellipsis: true}, {Page: 10, Ellipsis: false},
		}},
		{"second page still hugs the start", 2, 10, []pagerStep{
			{Page: 1, Ellipsis: false}, {Page: 2, Ellipsis: false}, {Page: 3, Ellipsis: false},
			{Page: 0, Ellipsis: true}, {Page: 10, Ellipsis: false},
		}},
		{"middle page is the 1 … 4 5 6 … N shape", 5, 10, []pagerStep{
			{Page: 1, Ellipsis: false}, {Page: 0, Ellipsis: true},
			{Page: 4, Ellipsis: false}, {Page: 5, Ellipsis: false}, {Page: 6, Ellipsis: false},
			{Page: 0, Ellipsis: true}, {Page: 10, Ellipsis: false},
		}},
		{"last page hugs the end", 10, 10, []pagerStep{
			{Page: 1, Ellipsis: false}, {Page: 0, Ellipsis: true},
			{Page: 8, Ellipsis: false}, {Page: 9, Ellipsis: false}, {Page: 10, Ellipsis: false},
		}},
		{"beyond range clamps to last", 99, 10, []pagerStep{
			{Page: 1, Ellipsis: false}, {Page: 0, Ellipsis: true},
			{Page: 8, Ellipsis: false}, {Page: 9, Ellipsis: false}, {Page: 10, Ellipsis: false},
		}},
		{"before range clamps to first", 0, 10, []pagerStep{
			{Page: 1, Ellipsis: false}, {Page: 2, Ellipsis: false}, {Page: 3, Ellipsis: false},
			{Page: 0, Ellipsis: true}, {Page: 10, Ellipsis: false},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pagerWindow(tt.page, tt.total)
			if len(got) != len(tt.want) {
				t.Fatalf("pagerWindow(%d, %d): got %d steps, want %d: %+v", tt.page, tt.total, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != (pagerStep{Page: tt.want[i].Page, Ellipsis: tt.want[i].Ellipsis}) {
					t.Fatalf("pagerWindow(%d, %d)[%d] = %+v, want %+v", tt.page, tt.total, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestPagerPageURL_PreservesState pins the state-preserving href contract:
// stamping the page keeps every other query param and replaces an existing
// page value rather than appending a second one.
func TestPagerPageURL_PreservesState(t *testing.T) {
	if got := pagerPageURL("/admin/users", 2); got != "/admin/users?page=2" {
		t.Errorf("bare base: got %q, want /admin/users?page=2", got)
	}
	if got := pagerPageURL("/admin/users?filter=mfa&q=a&sort=created", 2); got != "/admin/users?filter=mfa&page=2&q=a&sort=created" {
		t.Errorf("full state: got %q", got)
	}
	if got := pagerPageURL("/admin/users?page=5&q=a", 7); got != "/admin/users?page=7&q=a" {
		t.Errorf("existing page replaced: got %q", got)
	}
}

// TestPanel_UsersPagination seeds more users than one page and verifies the
// page window, the pagination footer, clamping beyond the last page, and the
// search+page combination (Phase 3 M053–M059).
func TestPanel_UsersPagination(t *testing.T) {
	ctx := context.Background()
	admin := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, admin)

	// Seed 60 extra users: 50 fill page one, 10 spill onto page two.
	for i := range 60 {
		email := fmt.Sprintf("pager%02d@p.dev", i)
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID:    identitymodel.SyntheticUserID("seed-" + email),
			Email: email,
		}); err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
	}

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	page1 := get("/admin/users")
	if page1.Code != http.StatusOK {
		t.Fatalf("page 1: status %d", page1.Code)
	}
	if !strings.Contains(page1.Body.String(), "page=2") {
		t.Error("page 1: pagination footer should link to page 2")
	}

	// Beyond-range clamps to the last page (still 200, not 404).
	page99 := get("/admin/users?page=99")
	if page99.Code != http.StatusOK {
		t.Fatalf("page 99 should clamp, got status %d", page99.Code)
	}

	// Search + page combine: a needle that only matches late-registered users
	// with an out-of-range page clamps to the single result page.
	combined := get("/admin/users?q=pager5&page=7")
	if combined.Code != http.StatusOK {
		t.Fatalf("search+page: status %d", combined.Code)
	}
	body := combined.Body.String()
	if !strings.Contains(body, "pager5") {
		t.Error("search+page: filtered rows should still render")
	}
}

// TestConfig_PageSizeClamped pins the Config.PageSize bounds: unset keeps the
// historical default, and anything outside MinPageSize..MaxListRows (10..200)
// snaps to the nearest boundary at New — a mistuned config must not shrink a
// page below the pager's design or balloon a single render.
func TestConfig_PageSizeClamped(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"unset keeps the default", 0, DefaultPageSize},
		{"below the floor snaps to it", 3, MinPageSize},
		{"exactly the floor", MinPageSize, MinPageSize},
		{"in range unchanged", 75, 75},
		{"at the ceiling", MaxListRows, MaxListRows},
		{"above the ceiling snaps to it", 5000, MaxListRows},
		{"negative snaps to the floor", -20, MinPageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel, err := New(Config{Service: svc, PageSize: tt.in})
			if err != nil {
				t.Fatalf("New(PageSize=%d): %v", tt.in, err)
			}
			if got := panel.config.PageSize; got != tt.want {
				t.Errorf("PageSize %d resolved to %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// TestPanel_PageSizeShapesList pins the Config.PageSize knob end-to-end
// through the users list: a page size of 10 splits an 11-user directory into
// two pages (10 rows, then the page=2 footer links the rest), while the
// default keeps the same directory on a single 50-row page — the knob changes
// rendering, the default changes nothing.
func TestPanel_PageSizeShapesList(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	// Seed 11 users: one page at the default 50, two pages at PageSize 10.
	for i := range 11 {
		email := fmt.Sprintf("size%02d@p.dev", i)
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID:    identitymodel.SyntheticUserID("seed-" + email),
			Email: email,
		}); err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
	}
	panelFor := func(t *testing.T, pageSize int) http.Handler {
		t.Helper()
		panel, err := New(Config{Service: svc, PageSize: pageSize, Authorizer: RequireAuthenticated()})
		if err != nil {
			t.Fatalf("New(PageSize=%d): %v", pageSize, err)
		}
		mux := http.NewServeMux()
		mw := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
			})
		}
		mux.Handle("/admin/", mw(http.StripPrefix("/admin", panel.Handler())))
		return mux
	}
	rows := func(body string) int {
		return strings.Count(body, `<a class="flex no-underline" href="/admin/users/`)
	}

	small := panelFor(t, 10)
	rec := httptest.NewRecorder()
	small.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("PageSize=10 page 1: status %d, body %s", rec.Code, rec.Body.String())
	}
	if got := rows(rec.Body.String()); got != 10 {
		t.Errorf("PageSize=10 page 1: %d rows, want 10", got)
	}
	if !strings.Contains(rec.Body.String(), "page=2") {
		t.Error("PageSize=10: pagination footer should link to page 2")
	}

	def := panelFor(t, 0) // 0 → the DefaultPageSize fallback
	rec2 := httptest.NewRecorder()
	def.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("default page: status %d", rec2.Code)
	}
	if got := rows(rec2.Body.String()); got != 11 {
		t.Errorf("default page size: %d rows on one page, want 11", got)
	}
	if strings.Contains(rec2.Body.String(), "page=2") {
		t.Error("default page size: 11 users must stay on one page")
	}
}

// seedDirectory registers n users with sortable ids so the row order is
// deterministic (the read model orders by CreatedAt, then id).
func seedDirectory(t *testing.T, svc *usermgmt.Service, n int, prefix string) {
	t.Helper()
	for i := range n {
		email := fmt.Sprintf("%s%02d@p.dev", prefix, i)
		if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
			ID:    identitymodel.SyntheticUserID(prefix + fmt.Sprintf("%03d", i)),
			Email: email,
		}); err != nil {
			t.Fatalf("register %s: %v", email, err)
		}
	}
}

// panelWithPageSize builds a panel at the given page size over svc (the
// TestPanel_PageSizeShapesList harness, extracted for the windowing tests).
func panelWithPageSize(t *testing.T, user *identitymodel.User, svc *usermgmt.Service, pageSize int) http.Handler {
	t.Helper()
	panel, err := New(Config{Service: svc, PageSize: pageSize, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatalf("New(PageSize=%d): %v", pageSize, err)
	}
	mux := http.NewServeMux()
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(usermgmt.WithUser(r.Context(), user)))
		})
	}
	mux.Handle("/admin/", mw(http.StripPrefix("/admin", panel.Handler())))
	return mux
}

// TestPanel_PagerWindowThroughPartial drives the windowing through the HTMX
// partial: a four-page list on page 2 renders 1 2 3 … 4 (first/last always
// linked, the dropped numbers an ellipsis, the current one marked), and the
// windowed links keep every other query param intact.
func TestPanel_PagerWindowThroughPartial(t *testing.T) {
	t.Parallel()

	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	seedDirectory(t, svc, 35, "win")
	h := panelWithPageSize(t, user, svc, 10)

	req := httptest.NewRequest(http.MethodGet, "/admin/users?sort=email&page=2", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("partial page 2: status %d", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"page=1", "page=3", "page=4", // first, neighbor, last — never dropped
		"page=2", // the current page's neighbors include itself in the window
		"…",      // the dropped middle numbers
		"sort=email",
		"aria-current=\"page\"",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("windowed partial missing %q", want)
		}
	}
	if strings.Contains(body, "page=5") {
		t.Error("four-page list must not link a page 5")
	}

	// A two-page list renders every number and no ellipsis at all.
	svc2, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	seedDirectory(t, svc2, 12, "two")
	h2 := panelWithPageSize(t, user, svc2, 10)

	req2 := httptest.NewRequest(http.MethodGet, "/admin/users?page=1", nil)
	req2.Header.Set("HX-Request", "true")
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("partial page 1: status %d", rec2.Code)
	}
	if strings.Contains(rec2.Body.String(), "…") {
		t.Error("a two-page window has nothing to elide")
	}
}
