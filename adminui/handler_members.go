package adminui

import (
	"net/http"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

// membersIndex is the tenant-scoped members page (ModeTenantAdmin).
func (h *Handler) membersIndex(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	var tenant *usermgmt.Tenant
	if t, err := h.cfg.Service.GetTenant(r.Context(), h.cfg.TenantID); err == nil {
		tenant = t
	}
	memberships := h.cfg.Service.TenantMembers(r.Context(), h.cfg.TenantID)
	members := make([]memberRow, 0, len(memberships))
	for _, m := range memberships {
		members = append(members, memberRow{Actor: m.ActorID.PrefixedString(), Roles: m.Roles})
	}
	p := h.page("Members", "/members", user)
	renderPage(w, r, membersPage(p, tenantDetailData{
		Tenant: tenant, Members: members, BasePath: h.cfg.BasePath,
	}))
}
