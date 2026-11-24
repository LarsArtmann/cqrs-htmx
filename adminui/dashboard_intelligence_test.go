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
	"github.com/larsartmann/templ-components/display"
)

// auditEntryAt builds a test audit entry with the given action and time.
func auditEntryAt(action string, at time.Time) usermgmt.AuditEntry {
	return usermgmt.AuditEntry{Action: action, OccurredAt: at}
}

// TestAuditSplitDelta_EmptyWindowIsNoDelta pins the honesty rule: a window
// with ZERO matching events cannot support a period-over-period claim, so
// ok=false and the card must render no change at all — never a fabricated
// "0% → 0%" stability delta.
func TestAuditSplitDelta_EmptyWindowIsNoDelta(t *testing.T) {
	cur, prev, ok := auditSplitDelta(nil, matchAll)
	if ok || cur != 0 || prev != 0 {
		t.Fatalf("empty window: got cur=%d prev=%d ok=%v, want all-zero with ok=false", cur, prev, ok)
	}
	change, trend := deltaChange(cur, prev, ok)
	if change != "" || trend != display.TrendNone {
		t.Fatalf("empty window delta = %q/%v, want no change and TrendNone", change, trend)
	}
}

// TestAuditSplitDelta_SplitsAtMidpoint pins the window math: matching events
// spread across a time span are split at the span's midpoint into later-half
// ("current") and earlier-half ("previous") counts.
func TestAuditSplitDelta_SplitsAtMidpoint(t *testing.T) {
	base := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	entries := []usermgmt.AuditEntry{
		auditEntryAt(usermgmt.AuditActionRegister, base.Add(0)),
		auditEntryAt(usermgmt.AuditActionRegister, base.Add(1*time.Hour)),
		auditEntryAt(usermgmt.AuditActionRegister, base.Add(2*time.Hour)),
		auditEntryAt(usermgmt.AuditActionRegister, base.Add(3*time.Hour)),
	}
	cur, prev, ok := auditSplitDelta(entries, matchAll)
	if !ok {
		t.Fatal("nonempty window must report ok")
	}
	if prev != 2 || cur != 2 {
		t.Fatalf("split = prev %d / cur %d, want 2 / 2", prev, cur)
	}
	change, trend := deltaChange(cur, prev, ok)
	if change != "±0" || trend != display.TrendNone {
		t.Fatalf("equal halves delta = %q/%v, want ±0/TrendNone", change, trend)
	}
}

// TestDeltaChange_Directions pins the sign vocabulary: growth reads "+N" with
// an up trend, decline reads "−N" with a down trend.
func TestDeltaChange_Directions(t *testing.T) {
	for _, tc := range []struct {
		cur, prev int
		wantText  string
		wantTrend display.TrendDirection
	}{
		{5, 3, "+2", display.TrendUp},
		{3, 5, "−2", display.TrendDown},
	} {
		change, trend := deltaChange(tc.cur, tc.prev, true)
		if change != tc.wantText || trend != tc.wantTrend {
			t.Errorf("delta(%d,%d) = %q/%v, want %q/%v", tc.cur, tc.prev, change, trend, tc.wantText, tc.wantTrend)
		}
	}
}

// TestDeltaChange_NonMatchingFilterIsHonest: a filter that matches nothing
// (e.g. zero login failures in a busy window) renders no delta — the card
// shows the count but refuses a trend it cannot support.
func TestDeltaChange_NonMatchingFilterIsHonest(t *testing.T) {
	base := time.Now().UTC()
	entries := []usermgmt.AuditEntry{auditEntryAt(usermgmt.AuditActionRegister, base)}
	cur, prev, ok := auditSplitDelta(entries, loginFailureMatch)
	if ok || cur != 0 || prev != 0 {
		t.Fatalf("no matches: cur=%d prev=%d ok=%v, want ok=false", cur, prev, ok)
	}
	card := loginFailureCard(entries)
	if card.Value != "0" || card.Change != "" {
		t.Fatalf("zero-failure card = %q/%q, want value 0 with no change text", card.Value, card.Change)
	}
	if card.Tone == display.StatToneYellow {
		t.Error("zero failures must not render the attention tone")
	}
}

// TestLoginFailureCard_ThresholdTone pins the amber rule: at the attention
// threshold the card switches to the yellow tone and shows a real delta.
func TestLoginFailureCard_ThresholdTone(t *testing.T) {
	base := time.Now().UTC()
	entries := make([]usermgmt.AuditEntry, 0, loginFailureAttentionThreshold)
	for i := range loginFailureAttentionThreshold {
		entries = append(entries, auditEntryAt(usermgmt.AuditActionLoginFailed, base.Add(time.Duration(i)*time.Minute)))
	}
	card := loginFailureCard(entries)
	if card.Value != "5" {
		t.Fatalf("value = %q, want 5", card.Value)
	}
	if card.Tone != display.StatToneYellow {
		t.Errorf("tone = %v at threshold, want yellow", card.Tone)
	}
	if card.Change == "" {
		t.Error("nonempty failure window must render a change")
	}
}

// TestRegistrationsPerDay_BucketsAndZeroDays pins the sparkline math: register
// entries bucket per local day inside the trailing range, non-register events
// are ignored, days without registrations stay an honest zero.
func TestRegistrationsPerDay_BucketsAndZeroDays(t *testing.T) {
	loc := time.UTC
	today := time.Now().In(loc)
	todayNoon := time.Date(today.Year(), today.Month(), today.Day(), 12, 0, 0, 0, loc)
	yesterdayNoon := todayNoon.AddDate(0, 0, -1)
	entries := []usermgmt.AuditEntry{
		auditEntryAt(usermgmt.AuditActionRegister, yesterdayNoon),
		auditEntryAt(usermgmt.AuditActionRegister, yesterdayNoon.Add(2*time.Hour)),
		auditEntryAt(usermgmt.AuditActionRegister, todayNoon),
		auditEntryAt(usermgmt.AuditActionLoginFailed, todayNoon), // ignored
	}
	points := registrationsPerDay(entries, loc, sparklineDays)
	if len(points) != sparklineDays {
		t.Fatalf("len = %d, want %d", len(points), sparklineDays)
	}
	if got := points[len(points)-1].Count; got != 1 {
		t.Errorf("today count = %d, want 1", got)
	}
	if got := points[len(points)-2].Count; got != 2 {
		t.Errorf("yesterday count = %d, want 2", got)
	}
	if got := points[0].Count; got != 0 {
		t.Errorf("oldest day count = %d, want honest 0", got)
	}
	if points[0].Day.After(points[len(points)-1].Day) {
		t.Error("points must run oldest → newest")
	}
}

// TestSparklinePoints_AllZeroIsFlatLine: a window with no registrations
// renders a flat baseline (all points on the same y), not a fabricated wave.
func TestSparklinePoints_AllZeroIsFlatLine(t *testing.T) {
	points := []dayCount{
		{Day: time.Unix(0, 0), Count: 0},
		{Day: time.Unix(0, 0), Count: 0},
		{Day: time.Unix(0, 0), Count: 0},
	}
	got := sparklinePoints(points)
	if !strings.Contains(got, ",38") {
		t.Fatalf("all-zero sparkline %q must sit on the flat baseline y=38", got)
	}
	if strings.Contains(got, ",37") || strings.Contains(got, ",36") || strings.Contains(got, ",4") {
		t.Errorf("all-zero sparkline %q must not vary vertically", got)
	}
}

// TestDashboardWindowCaptionText_HonestSpan pins the caption vocabulary: the
// caption states what the window holds and its exact from/to span; an empty
// window (no facts, no span) renders NO caption.
func TestDashboardWindowCaptionText_HonestSpan(t *testing.T) {
	withWindow := dashboardData{
		Window: auditWindowFacts{Holds: 7, Limit: 10_000, Dropped: 0},
		WindowSpan: auditWindowSpan{
			From:    time.Date(2026, 11, 24, 9, 5, 0, 0, time.UTC),
			To:      time.Date(2026, 11, 24, 16, 30, 0, 0, time.UTC),
			HasSpan: true,
		},
	}
	got := dashboardWindowCaptionText(withWindow)
	for _, want := range []string{"holds 7 events", "from Nov 24 09:05", "to Nov 24 16:30", "journal keeps the full history"} {
		if !strings.Contains(got, want) {
			t.Errorf("caption %q missing %q", got, want)
		}
	}
	empty := dashboardData{}
	if got := dashboardWindowCaptionText(empty); got != "" {
		t.Errorf("empty window caption = %q, want none", got)
	}
}

// TestAuditWindowSpanOf_EmptyWindowHasNoSpan: an empty window states no
// dates — HasSpan stays false so the caption cannot invent a range.
func TestAuditWindowSpanOf_EmptyWindowHasNoSpan(t *testing.T) {
	span := auditWindowSpanOf(nil)
	if span.HasSpan {
		t.Fatal("empty window must not claim a span")
	}
	at := time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)
	span = auditWindowSpanOf([]usermgmt.AuditEntry{auditEntryAt("x", at)})
	if !span.HasSpan || !span.From.Equal(at) || !span.To.Equal(at) {
		t.Fatalf("single-entry span = %v, want from=to=%v", span, at)
	}
}

// TestDashboard_RenderCarriesIntelligenceOnBothSurfaces is the polled-parity
// receipt: every derived element (window caption, login-failure card,
// operators card, sparkline) renders on the page AND in the polled stats
// partial, in super-admin mode.
func TestDashboard_RenderCarriesIntelligenceOnBothSurfaces(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	// Seed one registration so the audit window is non-empty and the
	// sparkline has a real point to draw.
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-intel"), Email: "intel@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ name, path string }{
		{"dashboard page", "/admin/"},
		{"polled stats partial", "/admin/partials/stats"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: status %d", tc.name, rec.Code)
			}
			html := rec.Body.String()
			for _, want := range []string{
				"Login failures",
				"Platform operators",
				"Registrations per day",
				"Audit window holds",
				"users?filter=admin",
			} {
				if !strings.Contains(html, want) {
					t.Errorf("%s missing %q", tc.name, want)
				}
			}
		})
	}
}

// TestDashboard_AdoptionCards pins the adoption ratios on the super-admin
// dashboard: verified, MFA, passkeys, OAuth cards all render, and a seeded
// passkey credential moves the passkey ratio off the empty "—" state.
// (External accounts have no public linking API, so the OAuth card is
// asserted on presence + honest empty value only.)
func TestDashboard_AdoptionCards(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "admin@example.com")
	h, svc := newTestPanel(t, user)

	reg, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: identitymodel.SyntheticUserID("u-adopt"), Email: "adopt@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddCredential(ctx, reg.User.ID, identitymodel.WebAuthnCredential{
		CredentialCore: identitymodel.CredentialCore{
			ID:        []byte("cred-1"),
			PublicKey: []byte("pk"),
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	html := rec.Body.String()
	for _, want := range []string{"Passkey adoption", "OAuth adoption", "Verified", "MFA adoption"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q", want)
		}
	}
	if !strings.Contains(html, "100%") {
		t.Error("single user with a passkey must read 100% passkey adoption")
	}
}

// TestDashboard_BusFactorBanner pins the banner variant matrix: no admin
// grant renders the stronger zero banner; exactly one operator renders the
// bus-factor banner; two or more operators render neither.
func TestDashboard_BusFactorBanner(t *testing.T) {
	ctx := context.Background()

	t.Run("single operator renders bus-factor banner", func(t *testing.T) {
		user := mustUser(t, "sole@example.com")
		h, svc := newTestPanel(t, user)
		if err := svc.GrantRoles(ctx,
			identitymodel.ActorIDFromUser(user.ID),
			identitymodel.NewTenantID(identitymodel.WildcardDomain),
			[]identitymodel.Role{identitymodel.RoleSuperAdmin},
		); err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		if html := rec.Body.String(); !strings.Contains(html, "Bus factor of one") {
			t.Error("single operator must render the bus-factor banner")
		}
	})

	t.Run("two operators render no bus-factor banner", func(t *testing.T) {
		user := mustUser(t, "first@example.com")
		h, svc := newTestPanel(t, user)
		second := mustUser(t, "second@example.com")
		for _, u := range []*identitymodel.User{user, second} {
			if err := svc.GrantRoles(ctx,
				identitymodel.ActorIDFromUser(u.ID),
				identitymodel.NewTenantID(identitymodel.WildcardDomain),
				[]identitymodel.Role{identitymodel.RoleAdmin},
			); err != nil {
				t.Fatal(err)
			}
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		if html := rec.Body.String(); strings.Contains(html, "Bus factor of one") {
			t.Error("two operators must not render the bus-factor banner")
		}
	})
}

// TestDashboard_OperatorsCardCountsDistinctAdminTier pins F075: the operators
// card counts DISTINCT users holding admin-tier (super_admin OR admin)
// wildcard grants — one user holding both roles counts once — and links to
// the platform-admin filtered users list.
func TestDashboard_OperatorsCardCountsDistinctAdminTier(t *testing.T) {
	ctx := context.Background()
	user := mustUser(t, "ops@example.com")
	ht, svc := newTestPanel(t, user)
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}
	// One user, BOTH admin-tier roles: distinct count must stay 1.
	for _, role := range []identitymodel.Role{identitymodel.RoleSuperAdmin, identitymodel.RoleAdmin} {
		if err := svc.GrantRoles(ctx,
			identitymodel.ActorIDFromUser(user.ID),
			identitymodel.NewTenantID(identitymodel.WildcardDomain),
			[]identitymodel.Role{role},
		); err != nil {
			t.Fatal(err)
		}
	}
	operators := panel.adminTierOperators()
	if len(operators) != 1 {
		t.Fatalf("distinct admin-tier operators = %d (%v), want 1", len(operators), operators)
	}
	rec := httptest.NewRecorder()
	ht.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	html := rec.Body.String()
	if !strings.Contains(html, "Platform operators") || !strings.Contains(html, "users?filter=admin") {
		t.Error("operators card with filtered-list link missing from stats partial")
	}
}
