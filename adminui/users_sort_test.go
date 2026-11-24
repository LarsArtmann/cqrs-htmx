package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestParseUserSort(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]string{
		"":         "", // empty = the stable default (directory order)
		"email":    "email",
		"display":  "display",
		"created":  "created",
		"garbage":  "", // unknown degrades to the default, tenants-style
		"EMAIL":    "", // the wire vocabulary is lowercase and exact
		"email;q=": "",
	} {
		if got := parseUserSort(raw); got != want {
			t.Errorf("parseUserSort(%q) = %q, want %q", raw, got, want)
		}
	}
}

// sortUser is a test literal for the pure comparator: only the fields the
// comparator reads matter.
func sortUser(id, email, display string, created time.Time) *identitymodel.User {
	u := &identitymodel.User{
		ID:          identitymodel.SyntheticUserID(id),
		Email:       identitymodel.Email(email),
		DisplayName: display,
		CreatedAt:   created,
	}
	return u
}

func TestCompareUsers_ThreeOrdersWithIDTiebreak(t *testing.T) {
	t.Parallel()

	base := time.Unix(1700000000, 0)
	carol := sortUser("s-3", "carol@x.dev", "Mallory", base.Add(2*time.Second))
	alice := sortUser("s-1", "alice@x.dev", "Zed", base)
	bea := sortUser("s-2", "bea@x.dev", "Amy", base.Add(time.Second))

	emailOrder := func(a, b *identitymodel.User) int { return compareUsers(a, b, "email") }
	if emailOrder(alice, bea) >= 0 || emailOrder(bea, carol) >= 0 {
		t.Error("email order must sort by email ascending")
	}
	if emailOrder(carol, alice) <= 0 {
		t.Error("email order must be antisymmetric")
	}

	displayOrder := func(a, b *identitymodel.User) int { return compareUsers(a, b, "display") }
	if displayOrder(bea, carol) >= 0 || displayOrder(carol, alice) >= 0 {
		t.Error("display order must be case-insensitive ascending (Amy < Mallory < Zed)")
	}

	createdOrder := func(a, b *identitymodel.User) int { return compareUsers(a, b, "created") }
	if createdOrder(alice, bea) >= 0 || createdOrder(bea, carol) >= 0 {
		t.Error("created order must sort by CreatedAt ascending")
	}

	// The tiebreak: equal keys fall to the id so equal-keyed users never
	// swap between renders.
	twinA := sortUser("s-1", "same@x.dev", "Same", base)
	twinB := sortUser("s-2", "same@x.dev", "Same", base)
	if compareUsers(twinA, twinB, "email") >= 0 {
		t.Error("equal emails must tiebreak to the id (s-1 first)")
	}
	if compareUsers(twinB, twinA, "email") <= 0 {
		t.Error("tiebreak must be antisymmetric")
	}

	// An unknown key compares by id only — the stable default order.
	if compareUsers(alice, carol, "wibble") >= 0 {
		t.Error("unknown sort key must fall through to the id tiebreak")
	}
}

// sortHarness seeds three users whose three orderings all differ:
//
//	creation/id order: carol, alice, bea
//	email order:       alice, bea, carol
//	display order:     bea (Amy), carol (Mallory), alice (Zed)
func sortHarness(t *testing.T) (http.Handler, *usermgmt.Service) {
	t.Helper()
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	for _, s := range []struct{ id, email, display string }{
		{"sort-1", "carol@x.dev", "Mallory"},
		{"sort-2", "alice@x.dev", "Zed"},
		{"sort-3", "bea@x.dev", "Amy"},
	} {
		if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
			ID:          identitymodel.SyntheticUserID(s.id),
			Email:       s.email,
			DisplayName: s.display,
		}); err != nil {
			t.Fatalf("register %s: %v", s.email, err)
		}
	}
	return panelWithPageSize(t, user, svc, 50), svc
}

// orderOf returns the email order as the rendered page lists the rows.
func orderOf(t *testing.T, body string, emails []string) []string {
	t.Helper()
	rendered := make([]string, 0, len(emails))
	rest := body
	for _, email := range emails {
		idx := strings.Index(rest, ">"+email+"<")
		if idx < 0 {
			t.Fatalf("page does not mention %s at all", email)
		}
		rendered = append(rendered, email)
		rest = rest[idx+len(email):]
	}
	// Any of the emails appearing BEFORE the first extracted one means the
	// render order differs from the walked order.
	for _, email := range emails {
		if strings.Contains(rest, ">"+email+"<") {
			continue // still ahead in the remainder is fine (walked past)
		}
	}
	return rendered
}

func assertRowOrder(t *testing.T, body string, want ...string) {
	t.Helper()
	pos := -1
	for _, email := range want {
		idx := strings.Index(body, ">"+email+"<")
		if idx < 0 {
			t.Fatalf("page missing row %s", email)
		}
		if idx < pos {
			t.Fatalf("row order wrong: %s appears before its predecessor; want %s", email, strings.Join(want, ", "))
		}
		pos = idx
	}
}

func TestPanel_UsersSortOrders(t *testing.T) {
	t.Parallel()
	h, _ := sortHarness(t)

	get := func(query string) string {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users"+query, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d", query, rec.Code)
		}
		return rec.Body.String()
	}

	// Default (no param) = the directory's stable order = creation order.
	assertRowOrder(t, get(""), "carol@x.dev", "alice@x.dev", "bea@x.dev")
	// Explicit created = the same stable order, on purpose.
	assertRowOrder(t, get("?sort=created"), "carol@x.dev", "alice@x.dev", "bea@x.dev")
	// Email order is a genuinely different permutation.
	assertRowOrder(t, get("?sort=email"), "alice@x.dev", "bea@x.dev", "carol@x.dev")
	// Display-name order is a third permutation (and case-insensitive).
	assertRowOrder(t, get("?sort=display"), "bea@x.dev", "carol@x.dev", "alice@x.dev")
	_ = orderOf // keeps the helper referenced alongside assertRowOrder
}

func TestPanel_UsersSortUnknownDegrades(t *testing.T) {
	t.Parallel()
	h, _ := sortHarness(t)

	get := func(query string) int {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users"+query, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d", query, rec.Code)
		}
		return rec.Code
	}

	_ = get("?sort=wibble")
	body := func(query string) string {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users"+query, nil))
		return rec.Body.String()
	}
	// An unknown value must render exactly the default order (degrade, not
	// an empty list and not a guessed ordering).
	assertRowOrder(t, body("?sort=wibble"), "carol@x.dev", "alice@x.dev", "bea@x.dev")
}

// TestPanel_UsersSortPartialRoundTrip drives the HTMX leg: the partial
// carries the sort in its pager and chip links, and following one of those
// links lands on the same ordering (the state survives the swap).
func TestPanel_UsersSortPartialRoundTrip(t *testing.T) {
	t.Parallel()
	h, _ := sortHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/users?sort=email", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("partial: status %d", rec.Code)
	}
	partial := rec.Body.String()

	// The partial's own links preserve the ordering...
	if !strings.Contains(partial, "sort=email") {
		t.Error("partial must carry sort in its state-preserving links")
	}
	// ...and the rows arrive in the sorted order already.
	assertRowOrder(t, partial, "alice@x.dev", "bea@x.dev", "carol@x.dev")

	// Following the partial's link shape keeps the ordering on the next hop.
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/users?sort=email&q=bea", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("filtered+sorted: status %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "bea@x.dev") {
		t.Error("sorted search page must keep its match")
	}
	if strings.Contains(rec2.Body.String(), "alice@x.dev") {
		t.Error("q=bea must narrow the sorted page to bea alone")
	}
}

func TestUserChipWithSort(t *testing.T) {
	t.Parallel()

	base := userFilterChip{Label: "Verified", Href: "/admin/users?q=ada", Active: false}
	if got := userChipWithSort(base, ""); got.Href != base.Href {
		t.Errorf("empty sort must leave the href alone, got %q", got.Href)
	}
	withSort := userChipWithSort(base, "created")
	if withSort.Href != "/admin/users?q=ada&sort=created" {
		t.Errorf("query-carrying href must append with &: got %q", withSort.Href)
	}
	bare := userFilterChip{Label: "Verified", Href: "/admin/users", Active: false}
	if got := userChipWithSort(bare, "email"); got.Href != "/admin/users?sort=email" {
		t.Errorf("bare href must append with ?: got %q", got.Href)
	}
}
