package adminui

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"slices"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// readPanelMemberCycle is the number of distinct member counts the read panel
// seeds: tenant i holds i%4 members, so the members sort has ties to break by
// display name.
const readPanelMemberCycle = 4

// memberReadCounter counts the tenants list's member-count reads by wrapping
// the Handler's read seam around the real service: one seam call is exactly
// one TenantMembers query.
type memberReadCounter struct {
	reads int
	// perTenant is the number of reads per tenant id.
	perTenant map[identitymodel.TenantID]int
}

// reset zeroes the counter so it measures a single request.
func (c *memberReadCounter) reset() {
	c.reads = 0
	clear(c.perTenant)
}

// readSlugs lists the tenants the counter saw, in slug order.
func (c *memberReadCounter) readSlugs() []string {
	slugs := make([]string, 0, len(c.perTenant))
	for id := range c.perTenant {
		slugs = append(slugs, id.Get())
	}
	slices.Sort(slugs)
	return slugs
}

// assertEachReadOnce fails unless exactly wantTenants distinct tenants were
// read and every one of them exactly once.
func assertEachReadOnce(t *testing.T, c *memberReadCounter, wantTenants int) {
	t.Helper()
	if len(c.perTenant) != wantTenants {
		t.Errorf("distinct tenants read = %d, want %d (%v)", len(c.perTenant), wantTenants, c.readSlugs())
	}
	for id, n := range c.perTenant {
		if n != 1 {
			t.Errorf("tenant %s read %d times, want exactly once", id.Get(), n)
		}
	}
}

// newTenantReadPanel builds a super-admin panel over n tenants named
// tenant-00 … tenant-{n-1}, where tenant i holds i%readPanelMemberCycle
// members (see the constant). It is served as an authenticated admin, and
// every member-count read the tenants list performs is counted.
func newTenantReadPanel(t *testing.T, n, pageSize int) (http.Handler, *memberReadCounter) {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	h, err := New(Config{Service: svc, Authorizer: RequireAuthenticated(), PageSize: pageSize})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	members := make([]identitymodel.UserID, 0, readPanelMemberCycle)
	for i := range readPanelMemberCycle {
		res := mustRegister(t, svc, "m"+strconv.Itoa(i), "m"+strconv.Itoa(i)+"@reads.dev")
		members = append(members, res.User.ID)
	}
	for i := range n {
		tenant := mustCreateTenant(t, svc, tenantSlug(i))
		for _, uid := range members[:i%readPanelMemberCycle] {
			mustAddMember(t, svc, uid, tenant.ID, identitymodel.RoleViewer)
		}
	}
	counter := &memberReadCounter{perTenant: map[identitymodel.TenantID]int{}}
	h.memberCount = func(ctx context.Context, tenantID identitymodel.TenantID) int {
		counter.reads++
		counter.perTenant[tenantID]++
		return len(svc.TenantMembers(ctx, tenantID))
	}
	return serveAs(h, mustUser(t, "admin@example.com")), counter
}

// tenantSlug is the id of the read panel's i-th tenant. Zero-padded, so slug,
// display name, and index order all agree.
func tenantSlug(i int) string { return fmt.Sprintf("tenant-%02d", i) }

// getTenants issues GET /admin/tenants{query} as the panel's admin. The full
// page is what a browser loads; the partial is the table fragment alone,
// without the page chrome (whose switcher lists every tenant).
func getTenants(t *testing.T, panel http.Handler, query string, partial bool) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants"+query, nil)
	if partial {
		req.Header.Set("HX-Request", "true")
	}
	panel.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/tenants%s: status %d", query, rec.Code)
	}
	return rec.Body.String()
}

// renderedTenantRows reads the table's rows out of a rendered body as
// "slug=count" tokens, in display order. The count link is the one element
// that carries both facts per row.
func renderedTenantRows(t *testing.T, body string) []string {
	t.Helper()
	countCell := regexp.MustCompile(`href="/admin/tenants/(tenant-\d+)">(\d+)\s*<span`)
	matches := countCell.FindAllStringSubmatch(body, -1)
	rows := make([]string, 0, len(matches))
	for _, m := range matches {
		rows = append(rows, m[1]+"="+m[2])
	}
	if len(rows) == 0 {
		t.Fatalf("no tenant rows in the rendered body: %.200s", body)
	}
	return rows
}

// rowsOf renders tenant indices as the "slug=count" tokens the read panel's
// tenants must show.
func rowsOf(indices []int) []string {
	rows := make([]string, 0, len(indices))
	for _, i := range indices {
		rows = append(rows, tenantSlug(i)+"="+strconv.Itoa(i%readPanelMemberCycle))
	}
	return rows
}

// membersOrder lists the read panel's tenant indices in members-sort order:
// member count descending, display name as the tiebreak.
func membersOrder(n int) []int {
	order := make([]int, 0, n)
	for i := range n {
		order = append(order, i)
	}
	slices.SortFunc(order, func(a, b int) int {
		if c := cmp.Compare(b%readPanelMemberCycle, a%readPanelMemberCycle); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	return order
}

// nameOrder lists the read panel's tenant indices in display-name order,
// which is index order for the zero-padded slugs.
func nameOrder(n int) []int {
	order := make([]int, 0, n)
	for i := range n {
		order = append(order, i)
	}
	return order
}

// pageWindow slices the rows of a 1-based page out of a full order.
func pageWindow(order []int, page, pageSize int) []int {
	start := min((page-1)*pageSize, len(order))
	return order[start:min(start+pageSize, len(order))]
}

// slugsOf names the given tenant indices.
func slugsOf(indices []int) []string {
	slugs := make([]string, 0, len(indices))
	for _, i := range indices {
		slugs = append(slugs, tenantSlug(i))
	}
	return slugs
}

// TestTenantsIndex_MemberCountReads pins the tenants list's read budget: a
// tenant's member count is read ONCE per request and reused for both the
// members sort and the rendered rows. The members sort ranks every MATCHED
// tenant (the order spans pages), so it costs one read per matched tenant —
// not the matched tenants plus the rendered page again; every other order
// reads only the rows it renders.
func TestTenantsIndex_MemberCountReads(t *testing.T) {
	t.Parallel()

	// One page (12 tenants fit the default 50) and three pages (23 tenants
	// at 10 per page). Before the one-pass fix these cost 24 (2N) and 33
	// (N + page) member-count reads respectively.
	layouts := []struct {
		name, was string
		tenants   int
		pageSize  int
	}{
		{name: "one page", was: "24 reads (2N)", tenants: 12, pageSize: DefaultPageSize},
		{name: "three pages", was: "33 reads (N + page)", tenants: 23, pageSize: 10},
	}
	for _, layout := range layouts {
		t.Run("members sort reads each matched tenant once: "+layout.name, func(t *testing.T) {
			panel, counter := newTenantReadPanel(t, layout.tenants, layout.pageSize)
			getTenants(t, panel, "?sort=members", false)
			if counter.reads != layout.tenants {
				t.Errorf("sort=members: member-count reads = %d, want %d (was %s)",
					counter.reads, layout.tenants, layout.was)
			}
			assertEachReadOnce(t, counter, layout.tenants)
		})
	}

	// The reused counts must still drive both the order and the cells: every
	// page shows its window of the members-sorted sequence with the right
	// count per row, and costs the same N reads as page one.
	t.Run("members sort keeps its order and counts on every page", func(t *testing.T) {
		const tenants, pageSize = 23, 10
		panel, counter := newTenantReadPanel(t, tenants, pageSize)
		order := membersOrder(tenants)
		for page := 1; page <= 3; page++ {
			counter.reset()
			body := getTenants(t, panel, fmt.Sprintf("?sort=members&page=%d", page), true)
			wantRows := rowsOf(pageWindow(order, page, pageSize))
			if got := renderedTenantRows(t, body); !slices.Equal(got, wantRows) {
				t.Errorf("page %d rows = %v, want %v", page, got, wantRows)
			}
			if counter.reads != tenants {
				t.Errorf("page %d: member-count reads = %d, want %d", page, counter.reads, tenants)
			}
			assertEachReadOnce(t, counter, tenants)
		}
	})

	// The members sort ranks the MATCHED tenants only — a search narrows
	// the reads together with the rows.
	t.Run("members sort reads only the search matches", func(t *testing.T) {
		panel, counter := newTenantReadPanel(t, 23, 10)
		getTenants(t, panel, "?q=tenant-1&sort=members", false)
		// "tenant-1" matches tenant-10 … tenant-19 and nothing else.
		want := slugsOf([]int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
		if counter.reads != len(want) {
			t.Errorf("search + sort=members: member-count reads = %d, want %d", counter.reads, len(want))
		}
		assertEachReadOnce(t, counter, len(want))
		if got := counter.readSlugs(); !slices.Equal(got, want) {
			t.Errorf("tenants read = %v, want %v", got, want)
		}
	})

	// Any other order must not inherit the members sort's cost: it reads
	// exactly the rows it renders — full page, then the short last page —
	// and keeps its display-name order and counts.
	t.Run("name sort reads only the rows it renders", func(t *testing.T) {
		const tenants, pageSize = 23, 10
		panel, counter := newTenantReadPanel(t, tenants, pageSize)
		order := nameOrder(tenants)
		for _, tc := range []struct{ page, wantReads int }{{1, 10}, {3, 3}} {
			counter.reset()
			body := getTenants(t, panel, fmt.Sprintf("?page=%d", tc.page), true)
			if counter.reads != tc.wantReads {
				t.Errorf("page %d: member-count reads = %d, want %d", tc.page, counter.reads, tc.wantReads)
			}
			assertEachReadOnce(t, counter, tc.wantReads)
			window := pageWindow(order, tc.page, pageSize)
			if got := counter.readSlugs(); !slices.Equal(got, slugsOf(window)) {
				t.Errorf("page %d tenants read = %v, want %v", tc.page, got, slugsOf(window))
			}
			if got := renderedTenantRows(t, body); !slices.Equal(got, rowsOf(window)) {
				t.Errorf("page %d rows = %v, want %v", tc.page, got, rowsOf(window))
			}
		}
	})
}
