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
