package adminui

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Query keys of the add-member error round trip. The typed email rides the
// redirect so the admin can correct it instead of retyping from scratch.
const (
	memberErrorParam   = "member_error"
	memberEmailParam   = "member_email"
	memberErrorUnknown = "unknown"
)

// lookupUser resolves a user id through the panel's service read model. It is
// the directory seam member rows use so a membership renders as a person.
func (h *Handler) lookupUser(id identitymodel.UserID) (*identitymodel.User, bool) {
	return h.config.Service.ReadModel().FindByUserID(id)
}

// applyMemberError fills the inline add-member guidance from the redirect
// query. Only the known code is honored: an unknown value renders nothing
// rather than echoing arbitrary text into the page.
func (h *Handler) applyMemberError(r *http.Request, d *tenantDetailData) {
	if r.URL.Query().Get(memberErrorParam) != memberErrorUnknown {
		return
	}
	email := strings.TrimSpace(r.URL.Query().Get(memberEmailParam))
	d.AddMemberEmail = email
	if email == "" {
		d.AddMemberError = "No account matches that email. Create the user first, then add them here."
		return
	}
	d.AddMemberError = "No account matches " + email +
		". Create the user first, then add them here."
}

// withMemberError appends the add-member error state to the return path.
func withMemberError(back, email string) string {
	u, err := url.Parse(back)
	if err != nil {
		return back
	}
	q := u.Query()
	q.Set(memberErrorParam, memberErrorUnknown)
	q.Set(memberEmailParam, email)
	u.RawQuery = q.Encode()
	return u.String()
}

// membersIndex is the tenant-scoped members page (ModeTenantAdmin).
func (h *Handler) membersIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	var tenant *usermgmt.Tenant
	if t, err := h.config.Service.GetTenant(r.Context(), h.config.TenantID); err == nil {
		tenant = t
	}
	allMembers := h.tenantRoster(r.Context(), h.config.TenantID)
	total := len(allMembers)
	// F157: the roster paginates beyond MaxListRows like every other list.
	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), total, h.config.PageSize)
	members := allMembers[offset : offset+limit]
	addedBy := h.addedByIndex(r.Context(), h.config.TenantID, members)
	for i := range members {
		members[i].AddedBy = addedBy[members[i].Actor.PrefixedString()]
	}
	memberBase := h.config.BasePath + "/members"
	// Tenant-admin mode serves NO user directory, so member rows carry no
	// detail links and the unknown-email banner offers no create escape hatch.
	d := tenantDetailData{
		Tenant:           tenant,
		Members:          members,
		AssignableRoles:  identitymodel.AssignableRoles(),
		BasePath:         h.config.BasePath,
		AddMemberURL:     memberBase,
		RemoveMemberBase: memberBase,
		UpdateRoleBase:   memberBase,
		TotalMembers:     total,
		AdminTierCount:   countAdminTier(allMembers),
		Page:             page,
		TotalPages:       totalPages,
		RosterURL:        memberBase,
		RosterExportURL:  memberBase + "/roster.csv",
	}
	h.applyMemberError(r, &d)
	p := h.page("Members", "/members", user, r)
	renderPage(w, r, membersPage(p, d))
}

// membersRosterCSV streams the scoped tenant's full roster (F156, tenant
// mode): the tenant-admin counterpart of the super-admin
// /tenants/{id}/roster.csv, sharing its writer and wire shape. Access
// mirrors GET /members exactly — the roster this mode can already see.
// The download filename carries the tenant id and the -export token, the
// same convention as every other export route (f#45 unification).
func (h *Handler) membersRosterCSV(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.writeRosterCSV(w, r, h.config.TenantID, "roster-export-"+h.config.TenantID.Get()+".csv")
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
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	roles, rolesOK := parseRolesForm(r)
	if email == "" || !rolesOK {
		triggerToast(w, "err", "Email and at least one role are required")
		redirect(w, r, back)
		return
	}
	target, ok := h.config.Service.ReadModel().FindByEmail(email)
	if !ok {
		triggerToast(w, "err", "No account matches that email")
		redirect(w, r, withMemberError(back, email))
		return
	}
	if err := h.config.Service.AddMember(
		r.Context(),
		identitymodel.ActorIDFromUser(target.ID),
		tenantID,
		roles,
	); err != nil {
		triggerToast(w, "err", "Add member failed: "+actionErrorMessage(err))
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
		triggerToast(w, "err", "Remove member failed: "+actionErrorMessage(err))
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

// lastAdminDowngradeRefused checks the DOWNGRADE half of the last-admin
// guard: when the actor currently holds an admin-tier role (admin or owner),
// the submitted role set drops it, and no OTHER member of the tenant holds
// an admin-tier role, the change would strand the tenant without anyone able
// to administer it. The membership decider
// (usermgmt/es_membership_decide.go decideUpdateMemberRoles) has no
// domain-level last-admin rule — it refuses only unknown memberships — so
// this panel-side guard is the only refusal, and it must refuse (not warn)
// on every surface that dispatches UpdateMemberRoles.
func (h *Handler) lastAdminDowngradeRefused(
	ctx context.Context, tenantID identitymodel.TenantID,
	actor identitymodel.ActorID, roles []identitymodel.Role,
) bool {
	if holdsAdminTier(roles) {
		return false // granting or keeping admin tier can never strand the tenant
	}
	adminTierCount, actorHoldsAdminTier := 0, false
	for _, m := range h.config.Service.TenantMembers(ctx, tenantID) {
		if holdsAdminTier(m.Roles) {
			adminTierCount++
		}
		if m.ActorID.PrefixedString() == actor.PrefixedString() {
			actorHoldsAdminTier = holdsAdminTier(m.Roles)
		}
	}
	return actorHoldsAdminTier && adminTierCount <= 1
}

func (h *Handler) doUpdateRole(
	w http.ResponseWriter, r *http.Request,
	tenantID identitymodel.TenantID, actor identitymodel.ActorID, back string,
) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	roles, ok := parseRolesForm(r)
	if !ok {
		triggerToast(w, "err", "Pick at least one role")
		redirect(w, r, back)
		return
	}
	if h.lastAdminDowngradeRefused(r.Context(), tenantID, actor, roles) {
		triggerToast(w, "err", lastAdminDowngradeRefusalMessage())
		redirect(w, r, back)
		return
	}
	if err := h.config.Service.UpdateMemberRoles(r.Context(), actor, tenantID, roles); err != nil {
		triggerToast(w, "err", "Role update failed: "+actionErrorMessage(err))
	} else {
		triggerToast(w, "ok", "Role updated")
	}
	redirect(w, r, back)
}
