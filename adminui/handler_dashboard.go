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
		stats = []statCard{
			{
				Label: "Users",
				Value: strconv.Itoa(svc.ReadModel().Count()),
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
