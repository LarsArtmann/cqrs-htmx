package adminui

import (
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// membersIndex is the tenant-scoped members page (ModeTenantAdmin).
func (h *Handler) membersIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
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
		AssignableRoles:  identitymodel.AssignableRoles(),
		BasePath:         h.config.BasePath,
		AddMemberURL:     memberBase,
		RemoveMemberBase: memberBase,
		UpdateRoleBase:   memberBase,
	}))
}

// membersAdd handles "add member" in tenant-admin mode (tenant from config).
func (h *Handler) membersAdd(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.doAddMember(w, r, h.config.TenantID, h.config.BasePath+"/members")
}

// membersRemove handles "remove member" in tenant-admin mode.
func (h *Handler) membersRemove(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	actor, ok := parseActorFromPath(w, r, h.config.BasePath+"/members")
	if !ok {
		return
	}
	h.doRemoveMember(w, r, h.config.TenantID, actor, h.config.BasePath+"/members")
}

// tenantAddMember handles "add member" in super-admin mode (tenant from path).
func (h *Handler) tenantAddMember(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	tenantID := identitymodel.NewTenantID(r.PathValue("id"))
	h.doAddMember(w, r, tenantID, h.config.BasePath+"/tenants/"+tenantID.Get())
}

// parseActorFromPath extracts and validates the actor ID from the URL path.
// Returns ok=false if the actor ID is missing or invalid.
func parseActorFromPath(w http.ResponseWriter, r *http.Request, back string) (identitymodel.ActorID, bool) {
	actor, err := identitymodel.ParseActorID(r.PathValue("actor"))
	if err != nil {
		triggerToast(w, "err", "Invalid actor ID")
		redirect(w, r, back)
		return identitymodel.ActorID{}, false
	}
	return actor, true
}

// parseTenantMemberPath extracts the tenant ID and actor ID from path values.
// Used by super-admin member handlers that take both from the URL.
// Returns ok=false if the actor ID is invalid.
func parseTenantMemberPath(
	w http.ResponseWriter, r *http.Request, back string,
) (identitymodel.TenantID, identitymodel.ActorID, bool) {
	actor, err := identitymodel.ParseActorID(r.PathValue("actor"))
	if err != nil {
		triggerToast(w, "err", "Invalid actor ID")
		redirect(w, r, back)
		return identitymodel.TenantID{}, identitymodel.ActorID{}, false
	}
	return identitymodel.NewTenantID(r.PathValue("id")), actor, true
}

// tenantRemoveMember handles "remove member" in super-admin mode.
func (h *Handler) tenantRemoveMember(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	// art-dupl:accept thin member handlers intentionally parallel: parse path, delegate to the action-specific do* method
	back := h.config.BasePath + "/tenants/" + r.PathValue("id")
	tenantID, actor, ok := parseTenantMemberPath(w, r, back)
	if !ok {
		return
	}
	h.doRemoveMember(w, r, tenantID, actor, back)
}

func (h *Handler) doAddMember(w http.ResponseWriter, r *http.Request, tenantID identitymodel.TenantID, back string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	role := identitymodel.Role(strings.TrimSpace(r.FormValue("role")))
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
		identitymodel.ActorIDFromUser(target.ID),
		tenantID,
		[]identitymodel.Role{role},
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
	tenantID identitymodel.TenantID,
	actor identitymodel.ActorID,
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
func (h *Handler) membersUpdateRole(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	actor, ok := parseActorFromPath(w, r, h.config.BasePath+"/members")
	if !ok {
		return
	}
	h.doUpdateRole(w, r, h.config.TenantID, actor, h.config.BasePath+"/members")
}

// tenantUpdateMemberRole handles "change role" in super-admin mode.
func (h *Handler) tenantUpdateMemberRole(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	// art-dupl:accept thin member handlers intentionally parallel: parse path, delegate to the action-specific do* method
	back := h.config.BasePath + "/tenants/" + r.PathValue("id")
	tenantID, actor, ok := parseTenantMemberPath(w, r, back)
	if !ok {
		return
	}
	h.doUpdateRole(w, r, tenantID, actor, back)
}

func (h *Handler) doUpdateRole(
	w http.ResponseWriter, r *http.Request,
	tenantID identitymodel.TenantID, actor identitymodel.ActorID, back string,
) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	role := identitymodel.Role(strings.TrimSpace(r.FormValue("role")))
	if role == "" {
		triggerToast(w, "err", "Role is required")
		redirect(w, r, back)
		return
	}
	if err := h.config.Service.UpdateMemberRoles(r.Context(), actor, tenantID, []identitymodel.Role{role}); err != nil {
		triggerToast(w, "err", "Role update failed: "+err.Error())
	} else {
		triggerToast(w, "ok", "Role updated")
	}
	redirect(w, r, back)
}
