package adminui

import (
	"net/http"
	"strconv"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/display"
)

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("Dashboard", "/", user, r)
	svc := h.config.Service

	var stats []statCard
	if h.config.Mode == ModeSuperAdmin {
		stats = []statCard{
			{Label: "Users", Value: strconv.Itoa(svc.ReadModel().Count()), Icon: iconUsers,
				Tone: display.StatToneBlue, Href: p.BasePath + "/users"},
			{Label: "Tenants", Value: strconv.Itoa(len(svc.AllTenants())), Icon: iconTenants,
				Tone: display.StatTonePurple, Href: p.BasePath + "/tenants"},
		}
	} else {
		members := svc.TenantMembers(r.Context(), h.config.TenantID)
		tenantName := h.config.TenantID.Get()
		if t, err := svc.GetTenant(r.Context(), h.config.TenantID); err == nil && t.DisplayName != "" {
			tenantName = t.DisplayName
		}
		stats = []statCard{
			{Label: "Members", Value: strconv.Itoa(len(members)), Icon: iconMembers,
				Tone: display.StatToneBlue, Href: p.BasePath + "/members"},
			{Label: "Tenant", Value: tenantName, Icon: iconTenants, Tone: display.StatTonePurple},
		}
	}
	if al := svc.AuditLog(); al != nil {
		stats = append(stats, statCard{
			Label: "Audit events", Value: strconv.Itoa(al.Count()), Icon: iconAudit,
			Tone: display.StatToneGreen, Href: p.BasePath + "/audit",
		})
	}

	var recent []usermgmt.AuditEntry
	if al := svc.AuditLog(); al != nil {
		recent = al.Recent(8)
		resolveAuditEmails(svc, recent)
	}

	renderPage(w, r, dashboardPage(p, dashboardData{Stats: stats, Recent: recent}))
}
