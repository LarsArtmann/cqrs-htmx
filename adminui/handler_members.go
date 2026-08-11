package adminui

import (
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// membersIndex is the tenant-scoped members page (ModeTenantAdmin).
func (h *Handler) membersIndex(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	var tenant *usermgmt.Tenant
	if t, err := h.config.Service.GetTenant(r.Context(), h.config.TenantID); err == nil {
		tenant = t
	}
	memberships := h.config.Service.TenantMembers(r.Context(), h.config.TenantID)
	members := toMemberRows(memberships)
	memberBase := h.config.BasePath + "/members"
	p := h.page("Members", "/members", user, r)
	renderPage(w, r, membersPage(p, tenantDetailData{
		Tenant:           tenant,
		Members:          members,
		AssignableRoles:  usermgmt.AssignableRoles(),
		BasePath:         h.config.BasePath,
		AddMemberURL:     memberBase,
		RemoveMemberBase: memberBase,
		UpdateRoleBase:   memberBase,
	}))
}

// membersAdd handles "add member" in tenant-admin mode (tenant from config).
func (h *Handler) membersAdd(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	h.doAddMember(w, r, h.config.TenantID, h.config.BasePath+"/members")
}

// membersRemove handles "remove member" in tenant-admin mode.
func (h *Handler) membersRemove(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	actor, ok := parseActorFromPath(w, r, h.config.BasePath+"/members")
	if !ok {
		return
	}
	h.doRemoveMember(w, r, h.config.TenantID, actor, h.config.BasePath+"/members")
}

// tenantAddMember handles "add member" in super-admin mode (tenant from path).
func (h *Handler) tenantAddMember(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	tenantID := usermgmt.NewTenantID(r.PathValue("id"))
	h.doAddMember(w, r, tenantID, h.config.BasePath+"/tenants/"+tenantID.Get())
}

// parseActorFromPath extracts and validates the actor ID from the URL path.
// Returns ok=false if the actor ID is missing or invalid.
func parseActorFromPath(w http.ResponseWriter, r *http.Request, back string) (usermgmt.ActorID, bool) {
	actor, err := usermgmt.ParseActorID(r.PathValue("actor"))
	if err != nil {
		triggerToast(w, "err", "Invalid actor ID")
		redirect(w, r, back)
		return usermgmt.ActorID{}, false
	}
	return actor, true
}

// parseTenantMemberPath extracts the tenant ID and actor ID from path values.
// Used by super-admin member handlers that take both from the URL.
// Returns ok=false if the actor ID is invalid.
func parseTenantMemberPath(
	w http.ResponseWriter, r *http.Request, back string,
) (usermgmt.TenantID, usermgmt.ActorID, bool) {
	actor, err := usermgmt.ParseActorID(r.PathValue("actor"))
	if err != nil {
		triggerToast(w, "err", "Invalid actor ID")
		redirect(w, r, back)
		return usermgmt.TenantID{}, usermgmt.ActorID{}, false
	}
	return usermgmt.NewTenantID(r.PathValue("id")), actor, true
}

// tenantRemoveMember handles "remove member" in super-admin mode.
func (h *Handler) tenantRemoveMember(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	back := h.config.BasePath + "/tenants/" + r.PathValue("id")
	tenantID, actor, ok := parseTenantMemberPath(w, r, back)
	if !ok {
		return
	}
	h.doRemoveMember(w, r, tenantID, actor, back)
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
	target, ok := h.config.Service.ReadModel().FindByEmail(email)
	if !ok {
		triggerToast(w, "err", "No user with that email")
		redirect(w, r, back)
		return
	}
	if err := h.config.Service.AddMember(
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
	if err := h.config.Service.RemoveMember(r.Context(), actor, tenantID); err != nil {
		triggerToast(w, "err", "Remove failed: "+err.Error())
	} else {
		triggerToast(w, "ok", "Member removed")
	}
	redirect(w, r, back)
}

// membersUpdateRole handles "change role" in tenant-admin mode.
func (h *Handler) membersUpdateRole(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	actor, ok := parseActorFromPath(w, r, h.config.BasePath+"/members")
	if !ok {
		return
	}
	h.doUpdateRole(w, r, h.config.TenantID, actor, h.config.BasePath+"/members")
}

// tenantUpdateMemberRole handles "change role" in super-admin mode.
func (h *Handler) tenantUpdateMemberRole(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	back := h.config.BasePath + "/tenants/" + r.PathValue("id")
	tenantID, actor, ok := parseTenantMemberPath(w, r, back)
	if !ok {
		return
	}
	h.doUpdateRole(w, r, tenantID, actor, back)
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
	if err := h.config.Service.UpdateMemberRoles(r.Context(), actor, tenantID, []usermgmt.Role{role}); err != nil {
		triggerToast(w, "err", "Role update failed: "+err.Error())
	} else {
		triggerToast(w, "ok", "Role updated")
	}
	redirect(w, r, back)
}
