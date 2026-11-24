package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// bulkHarness seeds a directory of three synthetic users plus the admin and
// wires a verification sender the tests can flip to per-address failure.
func bulkHarness(t *testing.T) (http.Handler, *usermgmt.Service, *sendRecorder, [3]string) {
	t.Helper()
	user := mustUser(t, "admin@example.com")
	rec := &sendRecorder{failFor: map[string]error{}}
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		AuditLog: usermgmt.NewAuditLog(),
		EmailVerification: &usermgmt.EmailVerificationConfig{
			SendEmail: func(_ context.Context, email, _ string) error {
				return rec.send(email)
			},
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	// The wire ids are the READ-MODEL id strings (SyntheticUserID seeds are
	// hashed), captured from each Register result — the tests select by the
	// same ids the directory actually serves.
	var ids [3]string
	for i, s := range []struct{ seed, email string }{
		{"bulk-1", "ada@x.dev"},
		{"bulk-2", "bea@x.dev"},
		{"bulk-3", "cy@x.dev"},
	} {
		res, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
			ID:    identitymodel.SyntheticUserID(s.seed),
			Email: s.email,
		})
		if err != nil {
			t.Fatalf("register %s: %v", s.email, err)
		}
		ids[i] = res.User.ID.Get().String()
	}
	return panelWithPageSize(t, user, svc, 50), svc, rec, ids
}

// sendRecorder lets a test name WHICH address fails, so the partial-failure
// panel can be asserted row by row.
type sendRecorder struct {
	failFor map[string]error
	sent    []string
}

func (r *sendRecorder) send(email string) error {
	if err, ok := r.failFor[email]; ok {
		return err
	}
	r.sent = append(r.sent, email)
	return nil
}

func bulkPost(h http.Handler, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/admin/users/bulk", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestPanel_UsersBulk_Resend_PerRowHonesty pins the F140 contract: one
// send failing does not blur the report — the panel names the succeeded
// row AND the refused row with its reason, counts exact.
func TestPanel_UsersBulk_Resend_PerRowHonesty(t *testing.T) {
	t.Parallel()
	h, _, rec, ids := bulkHarness(t)
	rec.failFor["bea@x.dev"] = errConfig("smtp refused the message")

	resp := bulkPost(h, url.Values{
		"ids":    []string{ids[0], ids[1]},
		"action": []string{"resend"},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("bulk resend: status %d body %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "1 succeeded, 1 refused") {
		t.Errorf("panel counts not exact: %s", body)
	}
	if !strings.Contains(body, "ada@x.dev") || !strings.Contains(body, "verification email sent") {
		t.Errorf("successful row not reported: %s", body)
	}
	// The refused row carries ITS OWN reason (whatever the service's error
	// mapping produces for the send failure) — never the success text, and
	// never a silent blur into the succeeded row.
	if !strings.Contains(body, "bea@x.dev") {
		t.Errorf("refused row not named: %s", body)
	}
	for line := range strings.SplitSeq(body, "<li>") {
		if strings.Contains(line, "bea@x.dev") && strings.Contains(line, "verification email sent") {
			t.Errorf("refused row must not read as sent: %s", line)
		}
	}
}

// TestPanel_UsersBulk_Resend_AllOK covers the clean path: no failure text,
// success count exact, and the sender actually called per row.
func TestPanel_UsersBulk_Resend_AllOK(t *testing.T) {
	t.Parallel()
	h, _, rec, ids := bulkHarness(t)

	resp := bulkPost(h, url.Values{
		"ids":    ids[:],
		"action": []string{"resend"},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("bulk resend: status %d", resp.Code)
	}
	if body := resp.Body.String(); !strings.Contains(body, "3 of 3 succeeded") {
		t.Errorf("all-ok panel wrong: %s", body)
	}
	if len(rec.sent) != 3 {
		t.Errorf("sender calls = %d, want 3", len(rec.sent))
	}
}

// TestPanel_UsersBulk_Grant_PlatformRole pins the grant path: the SAME
// durable GrantRoles dispatch the single-user roles route uses, with the
// panel's admin roles as the only wire vocabulary.
func TestPanel_UsersBulk_Grant_PlatformRole(t *testing.T) {
	t.Parallel()
	h, svc, _, ids := bulkHarness(t)

	resp := bulkPost(h, url.Values{
		"ids":    ids[:2],
		"action": []string{"grant"},
		"roles":  []string{string(identitymodel.RoleAdmin)},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("bulk grant: status %d body %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "2 of 2 succeeded") || !strings.Contains(body, "role granted") {
		t.Errorf("grant panel wrong: %s", body)
	}
	target, perr := identitymodel.ParseUserID(ids[0])
	if perr != nil {
		t.Fatalf("parse %s: %v", ids[0], perr)
	}
	roles, err := svc.Authz().RolesForUser(
		target,
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
	)
	if err != nil || len(roles) != 1 || roles[0] != identitymodel.RoleAdmin {
		t.Errorf("grant not durable in the policy engine: roles=%v err=%v", roles, err)
	}
}

// TestPanel_UsersBulk_RefusalPaths pins every honest no-op: no selection,
// an unknown action, and a role outside the admin set — each dispatches
// nothing and says so.
func TestPanel_UsersBulk_RefusalPaths(t *testing.T) {
	t.Parallel()
	h, _, rec, _ := bulkHarness(t)

	if resp := bulkPost(h, url.Values{"action": []string{"resend"}}); resp.Code != http.StatusOK ||
		!strings.Contains(resp.Body.String(), "No users selected") {
		t.Errorf("no-selection refusal wrong: %d %s", resp.Code, resp.Body.String())
	}
	resp := bulkPost(h, url.Values{"ids": []string{"ghost"}, "action": []string{"delete"}})
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "Pick an action") {
		t.Errorf("unknown-action refusal wrong: %d %s", resp.Code, resp.Body.String())
	}
	if resp := bulkPost(h, url.Values{
		"ids":    []string{"ghost"},
		"action": []string{"grant"},
		"roles":  []string{"super_boss"},
	}); resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "Pick a platform role") {
		t.Errorf("bad-role refusal wrong: %d %s", resp.Code, resp.Body.String())
	}
	if len(rec.sent) != 0 {
		t.Errorf("a refusal path dispatched a send: %v", rec.sent)
	}
}

// TestPanel_UsersSelectionExport_FiltersByIDs pins the ids-filtered export:
// only the checked users' rows, the Service export's wire shape, and the
// honest refusals (no ids / no survivors).
func TestPanel_UsersSelectionExport_FiltersByIDs(t *testing.T) {
	t.Parallel()
	h, _, _, ids := bulkHarness(t)

	selection := url.QueryEscape(ids[0] + "," + ids[2] + ",")
	req := httptest.NewRequest(http.MethodGet, "/admin/users/export.ids.csv?ids="+selection, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ids export: status %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "bea@x.dev") {
		t.Errorf("unchecked row leaked into the selection export: %s", body)
	}
	for _, want := range []string{"ada@x.dev", "cy@x.dev", "id,email,display_name"} {
		if !strings.Contains(body, want) {
			t.Errorf("selection export missing %q: %s", want, body)
		}
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "users-export-selection.csv") {
		t.Errorf("filename must say selection: %q", cd)
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/users/export.ids.csv", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "No users selected") {
		t.Errorf("empty-ids refusal wrong: %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/users/export.ids.csv?ids=ghost-1,ghost-2", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "None of the selected users exist") {
		t.Errorf("no-survivor refusal wrong: %d", rec.Code)
	}
}

// TestAvatarColorClass_StableAndInPalette pins the F183 contract: the same
// id always maps to the same class, and every class is inside the fixed
// six-swatch palette (never a computed color the CSS bundle cannot carry).
func TestAvatarColorClass_StableAndInPalette(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"01ABC", "01DEF", "different", ""} {
		first := avatarColorClass(id)
		if first != avatarColorClass(id) {
			t.Errorf("avatar class for %q is not stable: %q vs %q", id, first, avatarColorClass(id))
		}
		if !strings.HasPrefix(first, "admin-avatar-c") {
			t.Errorf("avatar class %q outside the fixed palette namespace", first)
		}
	}
	seen := map[string]bool{}
	for i := range 32 {
		seen[avatarColorClass("user-"+string(rune('a'+i)))] = true
	}
	if len(seen) < 3 {
		t.Errorf("hash spreads to only %d classes across 32 ids — palette unusable", len(seen))
	}
}

// TestParseUserSort_RecentAccepted pins the F184 wire vocabulary.
func TestParseUserSort_RecentAccepted(t *testing.T) {
	t.Parallel()
	if got := parseUserSort("recent"); got != "recent" {
		t.Errorf("parseUserSort(recent) = %q", got)
	}
	if got := parseUserSort("RECENT"); got != "" {
		t.Errorf("recent must be lowercase-exact, got %q", got)
	}
}

// TestSortUsersByLastSeen pins the F184 ordering: most recent first, users
// with no recorded activity last, and the id tiebreak keeping equal
// timestamps stable.
func TestSortUsersByLastSeen(t *testing.T) {
	t.Parallel()
	base := time.Unix(1700000000, 0)
	mallory := sortUser("s-3", "mallory@x.dev", "M", base)
	ada := sortUser("s-1", "ada@x.dev", "A", base)
	bea := sortUser("s-2", "bea@x.dev", "B", base)
	quiet := sortUser("s-4", "quiet@x.dev", "Q", base)
	// Keys are the READ-MODEL id strings (SyntheticUserID seeds are hashed),
	// captured from the users themselves — the same keys lastSeenIndex uses.
	lastSeen := map[string]time.Time{
		ada.ID.Get().String():     time.Unix(1700000300, 0), // most recent
		bea.ID.Get().String():     time.Unix(1700000100, 0),
		mallory.ID.Get().String(): time.Unix(1700000200, 0),
		// quiet: nothing recorded → sorts last
	}
	users := []*identitymodel.User{quiet, mallory, ada, bea}
	sortUsersByLastSeen(users, lastSeen)
	want := []*identitymodel.User{ada, mallory, bea, quiet}
	for i, u := range users {
		if u.ID.Get().String() != want[i].ID.Get().String() {
			t.Fatalf("order[%d] = %s, want %s", i, u.ID.Get().String(), want[i].ID.Get().String())
		}
	}

	// Equal timestamps tiebreak to the id, never reshuffle between renders.
	twinA := sortUser("s-a", "a@x.dev", "A", base)
	twinB := sortUser("s-b", "b@x.dev", "B", base)
	shared := map[string]time.Time{
		twinA.ID.Get().String(): base,
		twinB.ID.Get().String(): base,
	}
	pair := []*identitymodel.User{twinB, twinA}
	sortUsersByLastSeen(pair, shared)
	// The seed ids are hashed into ULIDs, so the expected winner is read off
	// the ids' own string form (the users_sort_test.go pattern), not assumed.
	wantFirst, wantSecond := twinA, twinB
	if strings.Compare(twinB.ID.Get().String(), twinA.ID.Get().String()) < 0 {
		wantFirst, wantSecond = twinB, twinA
	}
	if pair[0].ID.Get().String() != wantFirst.ID.Get().String() ||
		pair[1].ID.Get().String() != wantSecond.ID.Get().String() {
		t.Errorf("equal timestamps must tiebreak to the id: got %s before %s",
			pair[0].ID.Get().String(), pair[1].ID.Get().String())
	}
}
