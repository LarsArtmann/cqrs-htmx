package adminui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestParseAuditDateFilter(t *testing.T) {
	t.Parallel()

	if day, issue := parseAuditDateFilter("", "From"); day != nil || issue != "" {
		t.Errorf("empty bound = (%v, %q), want (nil, \"\")", day, issue)
	}
	day, issue := parseAuditDateFilter("garbage", "From")
	if day != nil {
		t.Error("malformed bound must yield no filter half")
	}
	if !strings.Contains(issue, "From") || !strings.Contains(issue, "ignored") {
		t.Errorf("malformed bound issue %q must name the input and the ignore", issue)
	}
	day, issue = parseAuditDateFilter("2026-11-01", "To")
	if issue != "" {
		t.Fatalf("valid bound produced issue %q", issue)
	}
	if !day.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)) {
		t.Errorf("valid bound parsed as %v, want local midnight of the given day", day)
	}
}

func TestFilterAuditRange_InclusiveWholeDays(t *testing.T) {
	t.Parallel()

	noon := time.Date(2026, 6, 15, 12, 0, 0, 0, time.Local)
	entries := []usermgmt.AuditEntry{
		{Action: "a", OccurredAt: noon},
		{Action: "b", OccurredAt: noon.AddDate(0, 0, 1)},
		{Action: "c", OccurredAt: noon.AddDate(0, 0, 5)},
	}

	day := func(y int, m time.Month, d int) *time.Time {
		tm := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		return &tm
	}

	if got := filterAuditRange(entries, nil, nil); len(got) != 3 {
		t.Errorf("no bounds filtered %d of 3 entries", len(got))
	}
	// from keeps its whole day: an entry at noon of the from-day stays.
	if got := filterAuditRange(entries, day(2026, 6, 15), nil); len(got) != 3 {
		t.Errorf("from-only range kept %d of 3 entries, want 3 (from day inclusive)", len(got))
	}
	if got := filterAuditRange(entries, day(2026, 6, 16), nil); len(got) != 2 {
		t.Errorf("from-after-first kept %d entries, want 2", len(got))
	}
	// to keeps its whole day: the to-day's noon entry stays.
	if got := filterAuditRange(entries, nil, day(2026, 6, 16)); len(got) != 2 {
		t.Errorf("to-only range kept %d entries, want 2 (to day inclusive)", len(got))
	}
	if got := filterAuditRange(entries, day(2026, 6, 16), day(2026, 6, 16)); len(got) != 1 {
		t.Errorf("single-day range kept %d entries, want 1", len(got))
	}
	if got := filterAuditRange(entries, day(2026, 6, 20), day(2026, 6, 25)); len(got) != 1 || got[0].Action != "c" {
		t.Errorf("late range kept %v, want only entry c", got)
	}
}

func TestParseAuditActorChip(t *testing.T) {
	t.Parallel()

	for _, kind := range auditActorChips {
		if got := parseAuditActorChip(kind); got != kind {
			t.Errorf("parseAuditActorChip(%q) = %q", kind, got)
		}
	}
	for _, raw := range []string{"", "weird", "USER", "users"} {
		if got := parseAuditActorChip(raw); got != "" {
			t.Errorf("unknown chip %q degraded to %q, want \"\" (every kind)", raw, got)
		}
	}
}

func TestFlow_AuditDateRangeAndIssues(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-range1"), Email: "range1@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-range2"), Email: "range2@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	get := func(url string) string {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", url, rec.Code)
		}
		return rec.Body.String()
	}

	// Unfiltered: both register rows visible.
	full := get("/admin/audit")
	if !strings.Contains(full, "range1@example.com") || !strings.Contains(full, "range2@example.com") {
		t.Error("unfiltered audit page missing seeded rows")
	}

	// Future from-bound: honest filtered-empty state, not the generic one.
	future := get("/admin/audit?from=2099-01-01")
	if strings.Contains(future, "range1@example.com") {
		t.Error("future from-bound did not filter rows out")
	}
	if !strings.Contains(future, "No events match your filters") {
		t.Error("filtered-empty state must say filters (not nothing recorded) caused it")
	}

	// to-only in the past: everything is after it, so nothing shows.
	past := get("/admin/audit?to=1999-01-01")
	if strings.Contains(past, "range1@example.com") {
		t.Error("past to-bound did not filter rows out")
	}

	// Reversed range: announced loudly AND ignored — rows stay visible.
	reversed := get("/admin/audit?from=2099-01-01&to=2098-01-01")
	if !strings.Contains(reversed, "From date is after To date") {
		t.Error("reversed range must render an explicit issue")
	}
	if !strings.Contains(reversed, "range1@example.com") {
		t.Error("a reversed range is ignored, so rows must stay visible")
	}

	// Malformed bound: issue names the input, that half is ignored.
	bad := get("/admin/audit?from=not-a-date")
	if !strings.Contains(bad, "From date is not a valid date") {
		t.Error("malformed from must render an issue naming the input")
	}
	if !strings.Contains(bad, "range1@example.com") {
		t.Error("a malformed bound half is ignored, so rows must stay visible")
	}

	// Filter state survives the round trip: inputs re-render with their
	// values and the chip hrefs carry the whole active filter set.
	dressed := get("/admin/audit?from=2099-01-01&to=2099-12-31&actor=user")
	if !strings.Contains(dressed, `value="2099-01-01"`) || !strings.Contains(dressed, `value="2099-12-31"`) {
		t.Error("date inputs must re-render with the submitted values")
	}
	if !strings.Contains(dressed, "from=2099-01-01") || !strings.Contains(dressed, "to=2099-12-31") {
		t.Error("chip/pager links must carry the active date range")
	}

	// HTMX partial: issues swap as a fragment, never a full page.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit?from=nope", nil)
	req.Header.Set("HX-Request", "true")
	h.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "<html") {
		t.Error("audit filter swap must be a fragment, not a full page")
	}
	if !strings.Contains(rec.Body.String(), "not a valid date") {
		t.Error("partial swap must keep the honest issue strip")
	}
}

func TestFlow_AuditActorChip(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-actor1"), Email: "actor1@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	al := svc.AuditLog()
	all := al.Recent(al.Count())
	if len(all) == 0 {
		t.Fatal("seed produced no audit entries")
	}
	present := map[string]bool{}
	for _, e := range all {
		present[e.ActorID.Kind().String()] = true
	}
	// A chip kind that actually acted keeps rows; a chip kind that never
	// acted shows the honest filtered-empty state; an unknown chip degrades
	// to every row and leaves only the All-actors chip active.
	var acted, absent string
	for _, kind := range auditActorChips {
		if acted == "" && present[kind] {
			acted = kind
		}
		if absent == "" && !present[kind] {
			absent = kind
		}
	}
	if acted == "" || absent == "" {
		t.Fatalf("seed kinds %v leave no (acted, absent) chip pair", present)
	}

	get := func(url string) string {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", url, rec.Code)
		}
		return rec.Body.String()
	}

	kept := get("/admin/audit?actor=" + acted)
	if strings.Count(kept, "<tr") < 2 {
		t.Errorf("actor chip %q (a kind that acted) must keep rows", acted)
	}
	none := get("/admin/audit?actor=" + absent)
	if strings.Count(none, "<tr") != 1 { // header row only
		t.Errorf("actor chip %q must filter every row out", absent)
	}
	if !strings.Contains(none, "No events match your filters") {
		t.Error("chip-filtered empty state must be honest about the filters")
	}
	degraded := get("/admin/audit?actor=weird")
	if strings.Count(degraded, "<tr") < 2 {
		t.Error("unknown chip must degrade to every row")
	}
	if strings.Contains(degraded, "actor=weird") {
		t.Error("unknown chip must leave no chip linking the weird value")
	}

	// The chip vocabulary renders: All + one per kind.
	for _, label := range []string{"All actors", "Users", "Bots", "System", "Services"} {
		if !strings.Contains(kept, label) {
			t.Errorf("audit page missing actor chip %q", label)
		}
	}
}

func TestRender_AuditWindowCaption(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-caption"), Email: "caption@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	al := svc.AuditLog()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit", nil))
	want := "Audit window holds " + strconv.Itoa(al.Count()) + " entries"
	if !strings.Contains(rec.Body.String(), want) {
		t.Errorf("audit page missing retention caption %q", want)
	}
	if !strings.Contains(rec.Body.String(), "the journal keeps the full history") {
		t.Error("retention caption must point at the durable record")
	}
}

func TestRender_AuditChangeIsHonestWhenPayloadAbsent(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	ada, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-change"), Email: "change@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeEmail(ctx, ada.User.ID, "changed@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeDisplayName(ctx, ada.User.ID, "Changed"); err != nil {
		t.Fatal(err)
	}

	// The audit projection records the change but not its old/new payload,
	// so the row says so — no fabricated diff is ever rendered.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit?q=changed", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "Email changed") || !strings.Contains(body, "Display name changed") {
		t.Error("audit page missing the change rows")
	}
	if !strings.Contains(body, "values not recorded") {
		t.Error("change rows without payload must say values are not recorded")
	}
	if strings.Contains(body, auditDiffOldClass) {
		t.Error("a fabricated diff (struck old value) must never render")
	}
}

func TestFlow_UserActivityLoadMore(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	target, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-activity"), Email: "activity@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 10; i++ {
		if err := svc.ChangeDisplayName(ctx, target.User.ID, fmt.Sprintf("Name-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	id := target.User.ID.Get().String()
	const total = 11 // register + 10 renames, all on the user's stream

	detail := renderUserDetail(t, h, id)
	if !strings.Contains(detail, `id="user-activity-more"`) {
		t.Error("user detail activity card missing the load-more row")
	}
	if !strings.Contains(detail, "/activity?skip=8") {
		t.Error("load-more row must fetch the next page at skip=8 (the card's page size)")
	}
	if strings.Contains(detail, "All "+strconv.Itoa(total)+" events in the audit window are shown.") {
		t.Error("end-of-window note must not render while rows remain unshown")
	}

	get := func(url string) string {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("HX-Request", "true")
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", url, rec.Code)
		}
		return rec.Body.String()
	}

	// The first load-more page exhausts this stream (8 shown + 8 more
	// requested of 11 total): the fragment carries the remaining rows and
	// the honest end note instead of another trigger.
	next := get("/admin/users/" + id + "/activity?skip=8")
	if strings.Contains(next, "<html") {
		t.Error("activity load-more must swap a fragment, not a full page")
	}
	if !strings.Contains(next, "All "+strconv.Itoa(total)+" events in the audit window are shown.") {
		t.Error("exhausted stream must render the end-of-window note with the true total")
	}
	if strings.Contains(next, "Load more") {
		t.Error("exhausted stream must not render another load-more trigger")
	}
	// Card page + one fragment together show every event exactly once: ten
	// rename rows (the register row rides the stream too, under its own
	// label).
	if got := strings.Count(detail+next, "Display name changed"); got != 10 {
		t.Errorf("card + fragment render %d rename rows, want 10", got)
	}

	// A mid-stream cursor leaves another page pending: the trigger rides
	// the fragment with the next skip.
	mid := get("/admin/users/" + id + "/activity?skip=2")
	if !strings.Contains(mid, "Load more") {
		t.Error("mid-stream fragment must carry the next load-more trigger")
	}

	// Tampered cursors degrade to the beginning (our own links emit the
	// cursor; a malformed one is tampering, not intent).
	for _, bad := range []string{"skip=abc", "skip=-5"} {
		frag := get("/admin/users/" + id + "/activity?" + bad)
		if !strings.Contains(frag, "Load more") {
			t.Errorf("%s must degrade to the first page (more rows pending)", bad)
		}
	}
	overrun := get("/admin/users/" + id + "/activity?skip=999")
	if !strings.Contains(overrun, "All "+strconv.Itoa(total)+" events in the audit window are shown.") {
		t.Error("overrun cursor must show the honest end note")
	}

	// A non-id path value refuses loudly.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users/not-an-id/activity", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid user id = %d, want 400", rec.Code)
	}
}
