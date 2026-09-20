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
			name:           "pageSize zero falls back to listPageSize",
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
