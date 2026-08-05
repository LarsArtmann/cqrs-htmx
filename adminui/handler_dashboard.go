package adminui

import (
	"net/http"
	"strconv"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	p := h.page("Dashboard", "/", user, r)
	svc := h.config.Service

	var stats []statCard
	if h.config.Mode == ModeSuperAdmin {
		stats = []statCard{
			{Label: "Users", Value: strconv.Itoa(svc.ReadModel().Count()), Icon: iconUsers},
			{Label: "Tenants", Value: strconv.Itoa(len(svc.AllTenants())), Icon: iconTenants},
		}
	} else {
		members := svc.TenantMembers(r.Context(), h.config.TenantID)
		tenantName := h.config.TenantID.Get()
		if t, err := svc.GetTenant(r.Context(), h.config.TenantID); err == nil && t.DisplayName != "" {
			tenantName = t.DisplayName
		}
		stats = []statCard{
			{Label: "Members", Value: strconv.Itoa(len(members)), Icon: iconMembers},
			{Label: "Tenant", Value: tenantName, Icon: iconTenants},
		}
	}
	if al := svc.AuditLog(); al != nil {
		stats = append(stats, statCard{
			Label: "Audit events", Value: strconv.Itoa(al.Count()), Icon: iconAudit,
		})
	}

	var recent []usermgmt.AuditEntry
	if al := svc.AuditLog(); al != nil {
		recent = al.Recent(8)
	}

	renderPage(w, r, dashboardPage(p, dashboardData{Stats: stats, Recent: recent}))
}
