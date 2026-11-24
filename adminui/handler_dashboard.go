package adminui

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
)

// dashboardWindowSizing bounds the window intelligence: how many days the
// registrations sparkline spans, and how many login failures in the window
// switch the failure card to the attention tone.
const (
	sparklineDays                 = 14
	loginFailureAttentionThreshold = 5
)

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("Dashboard", "/", user, r)
	d := dashboardData{
		Stats:    h.dashboardStats(r, p),
		StatsURL: p.BasePath + "/partials/stats",
		BasePath: p.BasePath,
	}

	if al := h.config.Service.AuditLog(); al != nil {
		d.Recent = al.Recent(8)
		resolveAuditEmails(h.config.Service, d.Recent)
	}

	// The bootstrap warning mirrors setup.New's boot diagnostic: when nobody
	// holds an admin-tier grant in the wildcard domain, the panel survives
	// only because a custom authorizer lets the caller in — a state worth
	// saying out loud before the last admin is gone.
	if has, err := h.config.Service.HasAdminGrant(
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
	); err == nil && !has {
		d.NoAdminGrant = true
	}

	// Bus factor: exactly one person holds every admin-tier wildcard grant,
	// so one lost credential or departure locks the platform out. The zero
	// case is the NoAdminGrant banner above; one is the bus-factor risk.
	d.BusFactor = len(h.adminTierOperators()) == 1

	// The window facts back the honesty caption and the sparkline; both are
	// polled-region elements, so the partial must carry them too.
	d.Window, d.WindowSpan, d.Registrations = h.dashboardWindowIntelligence()

	// superAdmin gates the quick-actions row: the create routes are
	// super-admin table rows, so tenant-admin mode must not render them.
	renderPage(w, r, dashboardPage(p, d, h.config.Mode == ModeSuperAdmin))
}

// statsPartial serves the polled stats region for htmx.PolledRegion
// (hx-swap=outerHTML): the response re-renders the region itself so polling
// continues across refreshes. Session-gated by the same guard as the page.
// The region carries the quick-actions row too, so a polled refresh keeps
// the page's row instead of dropping it (and the same mode gate applies).
func (h *Handler) statsPartial(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("Dashboard", "/", user, r)
	window, span, registrations := h.dashboardWindowIntelligence()
	renderPartial(w, r, dashboardStatsRegion(dashboardData{
		Stats:         h.dashboardStats(r, p),
		StatsURL:      p.BasePath + "/partials/stats",
		BasePath:      p.BasePath,
		Window:        window,
		WindowSpan:    span,
		Registrations: registrations,
	}, h.config.Mode == ModeSuperAdmin))
}

func (h *Handler) dashboardStats(r *http.Request, p pageData) []statCard {
	var stats []statCard
	if h.config.Mode == ModeSuperAdmin {
		stats = h.superAdminStats(p)
	} else {
		stats = h.tenantStats(r, p)
	}
	if al := h.config.Service.AuditLog(); al != nil {
		entries := al.Entries()
		change, trend := deltaChange(auditSplitDelta(entries, matchAll))
		stats = append(stats, statCard{
			Label:  "Audit events",
			Value:  strconv.Itoa(al.Count()),
			Icon:   iconAudit,
			Tone:   display.StatToneGreen,
			Href:   p.BasePath + "/audit",
			Change: change,
			Trend:  trend,
		})
		stats = append(stats, loginFailureCard(entries))
	}

	if s := h.config.SSEHubStats; s != nil {
		dropped := s.DroppedEvents()

		dropTone := display.StatToneGreen
		if dropped > 0 {
			dropTone = display.StatToneYellow
		}

		stats = append(stats,
			statCard{
				Label: "SSE live",
				Value: strconv.Itoa(s.SubscriberCount()),
				Icon:  iconSSE,
				Tone:  display.StatToneGreen,
			},
			statCard{
				Label: "Dropped events",
				Value: strconv.FormatInt(dropped, 10),
				Icon:  iconDrops,
				Tone:  dropTone,
			},
		)
	}

	return stats
}

// superAdminStats are the platform-wide cards: directory size, adoption
// ratios (verified, MFA, passkeys, OAuth), the operator count, and the
// tenant fleet with its suspension count.
func (h *Handler) superAdminStats(p pageData) []statCard {
	svc := h.config.Service
	users := svc.ReadModel().AllUsers()
	verified, mfa, passkey, oauth := 0, 0, 0, 0
	for _, u := range users {
		if u.EmailVerified {
			verified++
		}
		if u.TOTPEnabled {
			mfa++
		}
		if len(u.Credentials) > 0 {
			passkey++
		}
		if len(u.ExternalAccounts) > 0 {
			oauth++
		}
	}
	suspended := 0
	for _, t := range svc.AllTenants() {
		if t.Suspended {
			suspended++
		}
	}
	return []statCard{
		{
			Label: "Users",
			Value: strconv.Itoa(len(users)),
			Icon:  iconUsers,
			Tone:  display.StatToneBlue,
			Href:  p.BasePath + "/users",
		},
		{
			Label: labelVerified,
			Value: percent(verified, len(users)),
			Icon:  iconUsers,
			Tone:  display.StatToneBlue,
			Href:  p.BasePath + "/users",
		},
		{
			Label: "MFA adoption",
			Value: percent(mfa, len(users)),
			Icon:  iconUsers,
			Tone:  display.StatToneBlue,
			Href:  p.BasePath + "/users",
		},
		{
			Label: "Passkey adoption",
			Value: percent(passkey, len(users)),
			Icon:  iconUsers,
			Tone:  display.StatToneBlue,
			Href:  p.BasePath + "/users",
		},
		{
			Label: "OAuth adoption",
			Value: percent(oauth, len(users)),
			Icon:  iconUsers,
			Tone:  display.StatToneBlue,
			Href:  p.BasePath + "/users",
		},
		{
			Label: "Tenants",
			Value: strconv.Itoa(len(svc.AllTenants())),
			Icon:  iconTenants,
			Tone:  display.StatTonePurple,
			Href:  p.BasePath + "/tenants",
		},
		{
			Label: "Suspended tenants",
			Value: strconv.Itoa(suspended),
			Icon:  iconTenants,
			Tone:  display.StatTonePurple,
			Href:  p.BasePath + "/tenants",
		},
		{
			// Operators: everyone holding an admin-tier wildcard grant,
			// counted from Authz.UsersForRole. The link is the platform-admin
			// filtered users list — the same surface the roles chips drive.
			Label: "Platform operators",
			Value: strconv.Itoa(len(h.adminTierOperators())),
			Icon:  iconUsers,
			Tone:  display.StatToneBlue,
			Href:  p.BasePath + "/users?filter=admin",
		},
	}
}

// adminTierOperators lists the distinct users holding an admin-tier grant
// (super_admin or admin) in the wildcard domain, via Authz.UsersForRole —
// the same role-name-subject filter HasAdminGrant applies. A policy-engine
// failure yields an empty list, which the zero-grant banner already covers.
func (h *Handler) adminTierOperators() []string {
	authz := h.config.Service.Authz()
	if authz == nil {
		return nil
	}
	domain := identitymodel.NewTenantID(identitymodel.WildcardDomain)
	seen := make(map[string]struct{})
	for _, role := range []identitymodel.Role{identitymodel.RoleSuperAdmin, identitymodel.RoleAdmin} {
		ids, err := authz.UsersForRole(role, domain)
		if err != nil {
			continue
		}
		for _, id := range ids {
			seen[id] = struct{}{}
		}
	}
	operators := make([]string, 0, len(seen))
	for id := range seen {
		operators = append(operators, id)
	}
	slices.Sort(operators)
	return operators
}

// tenantStats are the tenant-mode cards: membership size and tenant identity.
func (h *Handler) tenantStats(r *http.Request, p pageData) []statCard {
	svc := h.config.Service
	members := svc.TenantMembers(r.Context(), h.config.TenantID)
	tenantName := h.config.TenantID.Get()
	if t, err := svc.GetTenant(r.Context(), h.config.TenantID); err == nil && t.DisplayName != "" {
		tenantName = t.DisplayName
	}
	return []statCard{
		{
			Label: "Members",
			Value: strconv.Itoa(len(members)),
			Icon:  iconMembers,
			Tone:  display.StatToneBlue,
			Href:  p.BasePath + "/members",
		},
		{Label: "Tenant", Value: tenantName, Icon: iconTenants, Tone: display.StatTonePurple},
	}
}

// percent renders part/total as an integer percentage; an empty set reads as
// "—" instead of a fake 0%.
func percent(part, total int) string {
	if total == 0 {
		return "—"
	}
	return strconv.Itoa(part*100/total) + "%"
}

// dashboardWindowIntelligence captures the audit window's facts, its
// oldest→newest reach, and the per-day registration counts that feed the
// sparkline. A missing audit log degrades to zero facts (no caption, no
// sparkline) — the panel runs without one by design.
func (h *Handler) dashboardWindowIntelligence() (auditWindowFacts, auditWindowSpan, []dayCount) {
	al := h.config.Service.AuditLog()
	if al == nil {
		return auditWindowFacts{}, auditWindowSpan{}, nil
	}
	entries := al.Entries()
	window := auditWindowFacts{Holds: al.Count(), Limit: al.Limit(), Dropped: al.Dropped()}
	return window, auditWindowSpanOf(entries), registrationsPerDay(entries, time.Now().Location(), sparklineDays)
}

// auditWindowSpanOf reads the window's temporal reach from its retained
// entries. An empty window has no span to state — HasSpan stays false so the
// caption cannot invent dates.
func auditWindowSpanOf(entries []usermgmt.AuditEntry) auditWindowSpan {
	if len(entries) == 0 {
		return auditWindowSpan{}
	}
	return auditWindowSpan{
		From:    entries[0].OccurredAt,
		To:      entries[len(entries)-1].OccurredAt,
		HasSpan: true,
	}
}

// auditSplitDelta splits the matched entries at the midpoint of their time
// span: events in the later half are "current", the earlier half "previous".
// ok is false when NOTHING matched — an empty selection cannot support an
// honest period-over-period comparison, so callers must render no delta at
// all rather than a fabricated stability claim.
func auditSplitDelta(entries []usermgmt.AuditEntry, match func(usermgmt.AuditEntry) bool) (cur, prev int, ok bool) {
	var matched []usermgmt.AuditEntry
	for _, e := range entries {
		if match(e) {
			matched = append(matched, e)
		}
	}
	if len(matched) == 0 {
		return 0, 0, false
	}
	oldest, newest := matched[0].OccurredAt, matched[len(matched)-1].OccurredAt
	cut := oldest.Add(newest.Sub(oldest) / 2)
	for _, e := range matched {
		if e.OccurredAt.After(cut) {
			cur++
		} else {
			prev++
		}
	}
	return cur, prev, true
}

// deltaChange formats a split delta as a stat card's Change/Trend pair. An
// unsupported comparison (ok=false, i.e. zero events) renders NOTHING — no
// change text, no trend arrow — because "0% → 0%" would claim a stability
// the window cannot attest. Equal nonzero counts honestly read "±0".
func deltaChange(cur, prev int, ok bool) (change string, trend display.TrendDirection) {
	if !ok {
		return "", display.TrendNone
	}
	switch {
	case cur > prev:
		return fmt.Sprintf("+%d", cur-prev), display.TrendUp
	case cur < prev:
		return fmt.Sprintf("−%d", prev-cur), display.TrendDown
	default:
		return "±0", display.TrendNone
	}
}

// loginFailureCard is the security-attention card: the window's LoginFailed
// count, amber (yellow tone) once it crosses the attention threshold. The
// delta carries the same honesty rule as every other card — an empty window
// renders no change.
func loginFailureCard(entries []usermgmt.AuditEntry) statCard {
	failures := 0
	for _, e := range entries {
		if e.Action == usermgmt.AuditActionLoginFailed {
			failures++
		}
	}
	tone := display.StatToneGreen
	if failures >= loginFailureAttentionThreshold {
		tone = display.StatToneYellow
	}
	change, trend := deltaChange(auditSplitDelta(entries, loginFailureMatch))
	return statCard{
		Label:  "Login failures",
		Value:  strconv.Itoa(failures),
		Icon:   iconDrops,
		Tone:   tone,
		Change: change,
		Trend:  trend,
	}
}

// matchAll selects every entry: the audit-volume card's delta compares the
// whole window against itself split in half.
func matchAll(usermgmt.AuditEntry) bool { return true }

// loginFailureMatch selects the window's failed-login entries for the delta
// split.
func loginFailureMatch(e usermgmt.AuditEntry) bool {
	return e.Action == usermgmt.AuditActionLoginFailed
}

// registrationsPerDay buckets the window's register entries per local day
// across a trailing fixed range ending today. Days with no registrations
// count 0 — the sparkline shows honest gaps, not a compressed timeline of
// only active days. Entries must be chronological (AuditLog.Entries order).
func registrationsPerDay(entries []usermgmt.AuditEntry, loc *time.Location, days int) []dayCount {
	if days <= 0 {
		return nil
	}
	today := time.Now().In(loc)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days - 1))
	counts := make(map[string]int, days)
	order := make([]dayCount, 0, days)
	for i := range days {
		day := start.AddDate(0, 0, i)
		order = append(order, dayCount{Day: day})
		counts[day.Format("2006-01-02")] = 0
	}
	for _, e := range entries {
		if e.Action != usermgmt.AuditActionRegister {
			continue
		}
		key := e.OccurredAt.In(loc).Format("2006-01-02")
		if _, inRange := counts[key]; inRange {
			counts[key]++
		}
	}
	for i := range order {
		order[i].Count = counts[order[i].Day.Format("2006-01-02")]
	}
	return order
}

// dashboardWindowCaptionText builds the honesty caption under the stat grid:
// what the window holds and the exact span its derived numbers cover. It is
// the dashboard sibling of the audit page's retention caption — same facts,
// same refusal to overclaim.
func dashboardWindowCaptionText(d dashboardData) string {
	if !d.Window.hasFacts() {
		return ""
	}
	caption := "Audit window holds " + strconv.Itoa(d.Window.Holds) + " events"
	if d.WindowSpan.HasSpan {
		caption += " from " + d.WindowSpan.From.Format("Jan 2 15:04") +
			" to " + d.WindowSpan.To.Format("Jan 2 15:04")
	}
	return caption + " · the journal keeps the full history."
}

// sparklinePoints maps the per-day counts into SVG polyline coordinates
// across a fixed 140×40 viewBox (2px vertical padding). A window whose
// maximum is zero (all days empty) renders a flat baseline — an honest
// picture of "nothing happened", not a fabricated wave.
func sparklinePoints(points []dayCount) string {
	max := 0
	for _, p := range points {
		if p.Count > max {
			max = p.Count
		}
	}
	n := len(points)
	coords := make([]string, 0, n)
	for i, p := range points {
		x := 0
		if n > 1 {
			x = i * 140 / (n - 1)
		}
		y := 38
		if max > 0 {
			y = 38 - p.Count*34/max
		}
		coords = append(coords, strconv.Itoa(x)+","+strconv.Itoa(y))
	}
	return strings.Join(coords, " ")
}

// sparklineAriaLabel is the text alternative (and visible caption) for the
// sparkline: total registrations across the days the window's range covers.
func sparklineAriaLabel(points []dayCount) string {
	total := 0
	for _, p := range points {
		total += p.Count
	}
	return fmt.Sprintf("%d registrations over %d days", total, len(points))
}
