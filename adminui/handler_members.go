package adminui

import (
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// membersIndex is the tenant-scoped members page (ModeTenantAdmin).
func (h *Handler) membersIndex(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	var tenant *usermgmt.Tenant
	if t, err := h.cfg.Service.GetTenant(r.Context(), h.cfg.TenantID); err == nil {
		tenant = t
	}
	memberships := h.cfg.Service.TenantMembers(r.Context(), h.cfg.TenantID)
	members := toMemberRows(memberships)
	memberBase := h.cfg.BasePath + "/members"
	p := h.page("Members", "/members", user, r)
	renderPage(w, r, membersPage(p, tenantDetailData{
		Tenant:           tenant,
		Members:          members,
		AssignableRoles:  usermgmt.AssignableRoles(),
		BasePath:         h.cfg.BasePath,
		AddMemberURL:     memberBase,
		RemoveMemberBase: memberBase,
		UpdateRoleBase:   memberBase,
	}))
}

// membersAdd handles "add member" in tenant-admin mode (tenant from config).
func (h *Handler) membersAdd(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	h.doAddMember(w, r, h.cfg.TenantID, h.cfg.BasePath+"/members")
}

// membersRemove handles "remove member" in tenant-admin mode.
func (h *Handler) membersRemove(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	actor := usermgmt.ParseActorID(r.PathValue("actor"))
	h.doRemoveMember(w, r, h.cfg.TenantID, actor, h.cfg.BasePath+"/members")
}

// tenantAddMember handles "add member" in super-admin mode (tenant from path).
func (h *Handler) tenantAddMember(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	tenantID := usermgmt.NewTenantID(r.PathValue("id"))
	h.doAddMember(w, r, tenantID, h.cfg.BasePath+"/tenants/"+tenantID.Get())
}

// parseTenantMemberPath extracts the tenant ID and actor ID from path values.
// Used by super-admin member handlers that take both from the URL.
func parseTenantMemberPath(r *http.Request) (usermgmt.TenantID, usermgmt.ActorID) {
	return usermgmt.NewTenantID(r.PathValue("id")), usermgmt.ParseActorID(r.PathValue("actor"))
}

// tenantRemoveMember handles "remove member" in super-admin mode.
func (h *Handler) tenantRemoveMember(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	tenantID, actor := parseTenantMemberPath(r)
	h.doRemoveMember(w, r, tenantID, actor, h.cfg.BasePath+"/tenants/"+tenantID.Get())
}

func (h *Handler) doAddMember(w http.ResponseWriter, r *http.Request, tenantID usermgmt.TenantID, back string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	role := usermgmt.Role(strings.TrimSpace(r.FormValue("role")))
	if email == "" || role == "" {
		triggerToast(w, "err", "Email and role are required")
		redirect(w, r, back)
		return
	}
	target, ok := h.cfg.Service.ReadModel().FindByEmail(email)
	if !ok {
		triggerToast(w, "err", "No user with that email")
		redirect(w, r, back)
		return
	}
	if err := h.cfg.Service.AddMember(
		r.Context(),
		usermgmt.ActorIDFromUser(target.ID),
		tenantID,
		[]usermgmt.Role{role},
	); err != nil {
		triggerToast(w, "err", "Add failed: "+err.Error())
	} else {
		triggerToast(w, "ok", "Member added")
	}
	redirect(w, r, back)
}

func (h *Handler) doRemoveMember(
	w http.ResponseWriter,
	r *http.Request,
	tenantID usermgmt.TenantID,
	actor usermgmt.ActorID,
	back string,
) {
	if err := h.cfg.Service.RemoveMember(r.Context(), actor, tenantID); err != nil {
		triggerToast(w, "err", "Remove failed: "+err.Error())
	} else {
		triggerToast(w, "ok", "Member removed")
	}
	redirect(w, r, back)
}

// membersUpdateRole handles "change role" in tenant-admin mode.
func (h *Handler) membersUpdateRole(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	actor := usermgmt.ParseActorID(r.PathValue("actor"))
	h.doUpdateRole(w, r, h.cfg.TenantID, actor, h.cfg.BasePath+"/members")
}

// tenantUpdateMemberRole handles "change role" in super-admin mode.
func (h *Handler) tenantUpdateMemberRole(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	tenantID, actor := parseTenantMemberPath(r)
	h.doUpdateRole(w, r, tenantID, actor, h.cfg.BasePath+"/tenants/"+tenantID.Get())
}

func (h *Handler) doUpdateRole(
	w http.ResponseWriter, r *http.Request,
	tenantID usermgmt.TenantID, actor usermgmt.ActorID, back string,
) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	role := usermgmt.Role(strings.TrimSpace(r.FormValue("role")))
	if role == "" {
		triggerToast(w, "err", "Role is required")
		redirect(w, r, back)
		return
	}
	if err := h.cfg.Service.UpdateMemberRoles(r.Context(), actor, tenantID, []usermgmt.Role{role}); err != nil {
		triggerToast(w, "err", "Role update failed: "+err.Error())
	} else {
		triggerToast(w, "ok", "Role updated")
	}
	redirect(w, r, back)
}
