package adminui

import (
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func (h *Handler) tenantsIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	tenants, total := capList(h.config.Service.AllTenants())
	d := tenantsListData{Tenants: tenants, Total: total, BasePath: h.config.BasePath}
	p := h.page("Tenants", "/tenants", user, r)
	renderPage(w, r, tenantsPage(p, d))
}

func (h *Handler) tenantNew(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("New tenant", "/tenants", user, r)
	renderPage(w, r, tenantNewPage(p, h.config.BasePath))
}

func (h *Handler) tenantCreate(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
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
	tenant, err := h.config.Service.CreateTenant(r.Context(), usermgmt.CreateTenantRequest{
		ID:          identitymodel.NewTenantID(name),
		Name:        name,
		DisplayName: display,
	})
	if err != nil {
		triggerToast(w, "err", "Create failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant created")
	redirect(w, r, h.config.BasePath+"/tenants/"+tenant.ID.Get())
}

func (h *Handler) tenantDetail(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	tenantID := identitymodel.NewTenantID(r.PathValue("id"))
	tenant, err := h.config.Service.GetTenant(r.Context(), tenantID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	memberships := h.config.Service.TenantMembers(r.Context(), tenantID)
	members := toMemberRows(memberships)
	memberBase := h.config.BasePath + "/tenants/" + tenantID.Get() + "/members"
	p := h.page(tenant.DisplayName, "/tenants", user, r)
	renderPage(w, r, tenantDetailPage(p, tenantDetailData{
		Tenant:           tenant,
		Members:          members,
		AssignableRoles:  identitymodel.AssignableRoles(),
		BasePath:         h.config.BasePath,
		AddMemberURL:     memberBase,
		RemoveMemberBase: memberBase,
		UpdateRoleBase:   memberBase,
	}))
}

func (h *Handler) tenantSuspend(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.SuspendTenant(r.Context(), id, "suspended via admin panel"); err != nil {
		triggerToast(w, "err", "Suspend failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant suspended")
	redirect(w, r, h.config.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantReactivate(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.ReactivateTenant(r.Context(), id); err != nil {
		triggerToast(w, "err", "Reactivate failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant reactivated")
	redirect(w, r, h.config.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantDelete(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.DeleteTenant(r.Context(), id, "deleted via admin panel"); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "Tenant deleted")
	redirect(w, r, h.config.BasePath+"/tenants")
}
