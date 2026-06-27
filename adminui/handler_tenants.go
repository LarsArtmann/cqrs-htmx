package adminui

import (
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

func (h *Handler) tenantsIndex(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	tenants, total := capList(h.cfg.Service.AllTenants())
	d := tenantsListData{Tenants: tenants, Total: total, BasePath: h.cfg.BasePath}
	p := h.page("Tenants", "/tenants", user, r)
	renderPage(w, r, tenantsPage(p, d))
}

func (h *Handler) tenantNew(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	p := h.page("New tenant", "/tenants", user, r)
	renderPage(w, r, tenantNewPage(p, h.cfg.BasePath))
}

func (h *Handler) tenantCreate(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	display := strings.TrimSpace(r.FormValue("display_name"))
	if display == "" {
		display = name
	}
	if name == "" {
		triggerToast(w, "err", "Tenant name is required")
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	tenant, err := h.cfg.Service.CreateTenant(r.Context(), usermgmt.CreateTenantRequest{
		ID:          usermgmt.NewTenantID(name),
		Name:        name,
		DisplayName: display,
	})
	if err != nil {
		triggerToast(w, "err", "Create failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant created")
	redirect(w, r, h.cfg.BasePath+"/tenants/"+tenant.ID.Get())
}

func (h *Handler) tenantDetail(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	tenantID := usermgmt.NewTenantID(r.PathValue("id"))
	tenant, err := h.cfg.Service.GetTenant(r.Context(), tenantID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	memberships := h.cfg.Service.TenantMembers(r.Context(), tenantID)
	members := make([]memberRow, 0, len(memberships))
	for _, m := range memberships {
		members = append(members, memberRow{Actor: m.ActorID, Roles: m.Roles})
	}
	memberBase := h.cfg.BasePath + "/tenants/" + tenantID.Get() + "/members"
	p := h.page(tenant.DisplayName, "/tenants", user, r)
	renderPage(w, r, tenantDetailPage(p, tenantDetailData{
		Tenant:           tenant,
		Members:          members,
		BasePath:         h.cfg.BasePath,
		AddMemberURL:     memberBase,
		RemoveMemberBase: memberBase,
	}))
}

func (h *Handler) tenantSuspend(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	id := usermgmt.NewTenantID(r.PathValue("id"))
	if err := h.cfg.Service.SuspendTenant(r.Context(), id, "suspended via admin panel"); err != nil {
		triggerToast(w, "err", "Suspend failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant suspended")
	redirect(w, r, h.cfg.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantReactivate(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	id := usermgmt.NewTenantID(r.PathValue("id"))
	if err := h.cfg.Service.ReactivateTenant(r.Context(), id); err != nil {
		triggerToast(w, "err", "Reactivate failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant reactivated")
	redirect(w, r, h.cfg.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantDelete(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	id := usermgmt.NewTenantID(r.PathValue("id"))
	if err := h.cfg.Service.DeleteTenant(r.Context(), id, "deleted via admin panel"); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant deleted")
	redirect(w, r, h.cfg.BasePath+"/tenants")
}
