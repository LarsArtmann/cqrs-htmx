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
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	display := strings.TrimSpace(r.FormValue("display_name"))
	if display == "" {
		display = name
	}
	if name == "" {
		triggerToast(w, "err", "Tenant name is required")
		h.writeErrorPage(w, r, http.StatusBadRequest, "Name required", "A tenant name is required.")
		return
	}
	tenant, err := h.config.Service.CreateTenant(r.Context(), usermgmt.CreateTenantRequest{
		ID:          identitymodel.NewTenantID(name),
		Name:        name,
		DisplayName: display,
	})
	if err != nil {
		triggerToast(w, "err", "Create failed: "+err.Error())
		h.writeErrorPage(w, r, http.StatusBadRequest, "Could not create tenant", err.Error())
		return
	}
	triggerToast(w, "ok", "Tenant created")
	redirect(w, r, h.config.BasePath+"/tenants/"+tenant.ID.Get())
}

func (h *Handler) tenantDetail(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	tenantID := identitymodel.NewTenantID(r.PathValue("id"))
	tenant, err := h.config.Service.GetTenant(r.Context(), tenantID)
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusNotFound,
			"Tenant not found",
			"No tenant with this id exists (it may have been deleted).",
		)
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
		h.writeErrorPage(w, r, http.StatusBadRequest, "Could not suspend tenant", err.Error())
		return
	}
	triggerToast(w, "ok", "Tenant suspended")
	redirect(w, r, h.config.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantReactivate(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.ReactivateTenant(r.Context(), id); err != nil {
		triggerToast(w, "err", "Reactivate failed: "+err.Error())
		h.writeErrorPage(w, r, http.StatusBadRequest, "Could not reactivate tenant", err.Error())
		return
	}
	triggerToast(w, "ok", "Tenant reactivated")
	redirect(w, r, h.config.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantDelete(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.DeleteTenant(r.Context(), id, "deleted via admin panel"); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		h.writeErrorPage(w, r, http.StatusBadRequest, "Could not delete tenant", err.Error())
		return
	}
	triggerToast(w, "ok", "Tenant deleted")
	redirect(w, r, h.config.BasePath+"/tenants")
}
