package adminui

import (
	"net/http"
	"strconv"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/templ-components/display"
)

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("Dashboard", "/", user, r)
	d := dashboardData{
		Stats:    h.dashboardStats(r, p),
		StatsURL: p.BasePath + "/partials/stats",
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

	renderPage(w, r, dashboardPage(p, d))
}

// statsPartial serves the polled stats region for htmx.PolledRegion
// (hx-swap=outerHTML): the response re-renders the region itself so polling
// continues across refreshes. Session-gated by the same guard as the page.
func (h *Handler) statsPartial(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("Dashboard", "/", user, r)
	renderPartial(w, r, dashboardStatsRegion(dashboardData{
		Stats:    h.dashboardStats(r, p),
		StatsURL: p.BasePath + "/partials/stats",
	}))
}

func (h *Handler) dashboardStats(r *http.Request, p pageData) []statCard {
	svc := h.config.Service
	var stats []statCard
	if h.config.Mode == ModeSuperAdmin {
		users := svc.ReadModel().AllUsers()
		verified, mfa := 0, 0
		for _, u := range users {
			if u.EmailVerified {
				verified++
			}
			if u.TOTPEnabled {
				mfa++
			}
		}
		suspended := 0
		for _, t := range svc.AllTenants() {
			if t.Suspended {
				suspended++
			}
		}
		stats = []statCard{
			{
				Label: "Users",
				Value: strconv.Itoa(len(users)),
				Icon:  iconUsers,
				Tone:  display.StatToneBlue,
				Href:  p.BasePath + "/users",
			},
			{
				Label: "Verified",
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
		}
	} else {
		members := svc.TenantMembers(r.Context(), h.config.TenantID)
		tenantName := h.config.TenantID.Get()
		if t, err := svc.GetTenant(r.Context(), h.config.TenantID); err == nil && t.DisplayName != "" {
			tenantName = t.DisplayName
		}
		stats = []statCard{
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
	if al := svc.AuditLog(); al != nil {
		stats = append(stats, statCard{
			Label: "Audit events",
			Value: strconv.Itoa(al.Count()),
			Icon:  iconAudit,
			Tone:  display.StatToneGreen,
			Href:  p.BasePath + "/audit",
		})
	}
	return stats
}

// percent renders part/total as an integer percentage; an empty set reads as
// "—" instead of a fake 0%.
func percent(part, total int) string {
	if total == 0 {
		return "—"
	}
	return strconv.Itoa(part*100/total) + "%"
}
