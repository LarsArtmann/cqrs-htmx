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
		Email:       email,
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
	// swap between renders. The ids are ULID-hashed, so the expected order
	// is read off the ids' own string form rather than assumed.
	twinA := sortUser("s-1", "same@x.dev", "Same", base)
	twinB := sortUser("s-2", "same@x.dev", "Same", base)
	wantSign := strings.Compare(twinA.ID.Get().String(), twinB.ID.Get().String())
	gotSign := compareUsers(twinA, twinB, "email")
	if gotSign != wantSign || gotSign == 0 {
		t.Errorf("equal emails must tiebreak to the id: got %d, want %d", gotSign, wantSign)
	}

	// An unknown key compares by id only — the stable default order.
	if compareUsers(alice, carol, "wibble") != strings.Compare(alice.ID.Get().String(), carol.ID.Get().String()) {
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

// orderOf reports the email order as the rendered page lists the rows —
// the positions of each row's email text must be strictly increasing.
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
}

func TestPanel_UsersSortUnknownDegrades(t *testing.T) {
	t.Parallel()
	h, _ := sortHarness(t)

	body := func(query string) string {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users"+query, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d", query, rec.Code)
		}
		return rec.Body.String()
	}
	// An unknown value must render exactly the default order (degrade, not
	// an empty list and not a guessed ordering).
	assertRowOrder(t, body("?sort=wibble"), "carol@x.dev", "alice@x.dev", "bea@x.dev")
}

// TestPanel_UsersSortPartialRoundTrip drives the HTMX leg: with two pages,
// the partial's pager and sort links carry the ordering, and following that
// link shape lands on the same ordering (the state survives the swap).
func TestPanel_UsersSortPartialRoundTrip(t *testing.T) {
	t.Parallel()
	user := mustUser(t, "admin@example.com")
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	seedDirectory(t, svc, 12, "srt") // emails srt00@… srt11@…: email order == id order
	h := panelWithPageSize(t, user, svc, 10)

	req := httptest.NewRequest(http.MethodGet, "/admin/users?sort=email", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("partial: status %d", rec.Code)
	}
	partial := rec.Body.String()

	// The partial's pager keeps the ordering next to the page param...
	if !strings.Contains(partial, "sort=email") {
		t.Error("partial must carry sort in its state-preserving pager links")
	}
	if !strings.Contains(partial, "page=2") {
		t.Error("two-page list must offer page 2 from the partial")
	}
	// ...and page one shows the first ten emails in sorted order.
	assertRowOrder(t, partial, "srt00@p.dev", "srt05@p.dev", "srt09@p.dev")
	if strings.Contains(partial, "srt10@p.dev") {
		t.Error("page one must not spill page-two rows")
	}

	// Following the partial's own link shape keeps the ordering on page two.
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/users?sort=email&page=2", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("sorted page 2: status %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "srt10@p.dev") || !strings.Contains(rec2.Body.String(), "srt11@p.dev") {
		t.Error("sorted page 2 must carry the last two emails")
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
