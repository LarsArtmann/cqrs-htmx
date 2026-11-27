package adminui

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
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
func (c *memberReadCounter) assertEachReadOnce(t *testing.T, wantTenants int) {
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
		res := mustRegister(t, svc, fmt.Sprintf("m%d", i), fmt.Sprintf("m%d@reads.dev", i))
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

// getTenantsTable issues GET /admin/tenants{query} as an HTMX partial and
// returns the rendered table fragment — the rows alone, without the page
// chrome (whose switcher lists every tenant).
func getTenantsTable(t *testing.T, panel http.Handler, query string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/tenants"+query, nil)
	req.Header.Set("HX-Request", "true")
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
	countCell := regexp.MustCompile(`href="/admin/tenants/(tenant-\d+)">(\d+) <span`)
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

// expectedTenantRows renders tenant indices as the "slug=count" tokens the
// read panel's tenants must show.
func expectedTenantRows(indices []int) []string {
	rows := make([]string, 0, len(indices))
	for _, i := range indices {
		rows = append(rows, fmt.Sprintf("%s=%d", tenantSlug(i), i%readPanelMemberCycle))
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

// nameOrder lists the read panel's tenant indices in display-name order.
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

func slugs(indices []int) []string {
	out := make([]string, 0, len(indices))
	for _, i := range indices {
		out = append(out, tenantSlug(i))
	}
	return out
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
		name, note string
		tenants    int
		pageSize   int
	}{
		{name: "one page", note: "was 24 reads (2N)", tenants: 12, pageSize: DefaultPageSize},
		{name: "three pages", note: "was 33 reads (N + page)", tenants: 23, pageSize: 10},
	}

	for _, layout := range layouts {
		t.Run("members sort reads each matched tenant once: "+layout.name, func(t *testing.T) {
			panel, counter := newTenantReadPanel(t, layout.tenants, layout.pageSize)
			getTenantsTable(t, panel, "?sort=members")
			if counter.reads != layout.tenants {
				t.Errorf("sort=members: member-count reads = %d, want %d (%s)",
					counter.reads, layout.tenants, layout.note)
			}
			counter.assertEachReadOnce(t, layout.tenants)
		})
	}

	// The reused counts must still drive both the order and the cells: every
	// page shows the same members-sorted sequence and the right count per
	// row, and costs the same N reads as page one.
	t.Run("members sort keeps its order and counts on every page", func(t *testing.T) {
		const tenants, pageSize = 23, 10
		panel, counter := newTenantReadPanel(t, tenants, pageSize)
		order := membersOrder(tenants)
		for page, wantRows := range [][]string{
			expectedTenantRows(pageWindow(order, 1, pageSize)),
			expectedTenantRows(pageWindow(order, 2, pageSize)),
			expectedTenantRows(pageWindow(order, 3, pageSize)),
		} {
			getTenantsTable(t, panel, fmt.Sprintf("?sort=members&page=%d", page+1))
			if got := renderedTenantRows(t, getTenantsTable(t, panel, fmt.Sprintf("?sort=members&page=%d", page+1))); !slices.Equal(got, wantRows) {
				t.Errorf("page %d rows = %v, want %v", page+1, got, wantRows)
			}
			if counter.reads != tenants {
				t.Errorf("page %d: member-count reads = %d, want %d", page+1, counter.reads, tenants)
			}
			counter.assertEachReadOnce(t, tenants)
		}
	})

	// The members sort ranks the MATCHED tenants only — a search narrows
	// the reads with the rows.
	t.Run("members sort reads only the search matches", func(t *testing.T) {
		const tenants, pageSize = 23, 10
		panel, counter := newTenantReadPanel(t, tenants, pageSize)
		getTenantsTable(t, panel, "?q=tenant-1&sort=members")
		want := []string{
			"tenant-10", "tenant-11", "tenant-12", "tenant-13", "tenant-14",
			"tenant-15", "tenant-16", "tenant-17", "tenant-18", "tenant-19",
		}
		if counter.reads != len(want) {
			t.Errorf("search + sort=members: member-count reads = %d, want %d", counter.reads, len(want))
		}
		if got := counter.readSlugs(); !slices.Equal(got, want) {
			t.Errorf("tenants read = %v, want %v", got, want)
		}
	})

	// Any other order must not pay for the members sort: it reads exactly
	// the rows it renders, and each of them once.
	t.Run("name sort reads only the visible rows", func(t *testing.T) {
		const tenants, pageSize = 23, 10
		for _, tc := range []struct {
			page, wantReads int
		}{{page: 1, wantReads: 10}, {page: 3, wantReads: 3}} {
			panel, counter := newTenantReadPanel(t, tenants, pageSize)
			getTenantsTable(t, panel, fmt.Sprintf("?page=%d", tc.page))
			if counter.reads != tc.wantReads {
				t.Errorf("page %d: member-count reads = %d, want %d", tc.page, counter.reads, tc.wantReads)
			}
			counter.assertEachReadOnce(t, tc.wantReads)
			want := slugs(pageWindow(nameOrder(tenants), tc.page, pageSize))
			if got := counter.readSlugs(); !slices.Equal(got, want) {
				t.Errorf("page %d tenants read = %v, want %v", tc.page, got, want)
			}
		}
	})
}

// TestTenantsIndex_NameSortRowsUnchanged pins the default order the read
// budget must not disturb: display-name order, windowed per page, with the
// right count in every cell.
func TestTenantsIndex_NameSortRowsUnchanged(t *testing.T) {
	t.Parallel()

	const tenants, pageSize = 23, 10
	panel, _ := newTenantReadPanel(t, tenants, pageSize)
	order := nameOrder(tenants)
	for page, wantRows := range [][]string{
		expectedTenantRows(pageWindow(order, 1, pageSize)),
		expectedTenantRows(pageWindow(order, 2, pageSize)),
		expectedTenantRows(pageWindow(order, 3, pageSize)),
	} {
		got := renderedTenantRows(t, getTenantsTable(t, panel, fmt.Sprintf("?page=%d", page+1)))
		if !slices.Equal(got, wantRows) {
			t.Errorf("page %d rows = %v, want %v", page+1, got, wantRows)
		}
	}
}

// TestTenantsIndex_MemberSortLinksUnchanged pins the sort links the list
// renders, so the reuse cannot shift the toggle's URLs.
func TestTenantsIndex_MemberSortLinksUnchanged(t *testing.T) {
	t.Parallel()

	panel, _ := newTenantReadPanel(t, 12, DefaultPageSize)
	body := getTenantsTable(t, panel, "?sort=members")
	if !strings.Contains(body, `/admin/tenants?sort=members`) {
		t.Error("members-sorted table must keep the sort=members link")
	}
	if !strings.Contains(body, `/admin/tenants"`) || strings.Contains(body, `/admin/tenants?sort=name`) {
		t.Error("name-sort link must stay the canonical /admin/tenants URL")
	}
}
