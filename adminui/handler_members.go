package adminui

import (
	"net/http"
	"net/url"
	"slices"
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

// memberEmailOptions lists the directory's email addresses for the add-member
// autocomplete. Sorted and bounded (MaxListRows) so a large directory cannot
// balloon the page; the field stays a free-text email input, so typing an
// address outside the list still works.
func (h *Handler) memberEmailOptions(exclude []memberRow) []string {
	alreadyMember := make(map[string]bool, len(exclude))
	for _, m := range exclude {
		if m.Email != "" {
			alreadyMember[strings.ToLower(m.Email)] = true
		}
	}
	all := h.config.Service.ReadModel().AllUsers()
	emails := make([]string, 0, min(len(all), MaxListRows))
	for _, u := range all {
		// A duplicate add is refused by the decider, so offering an existing
		// member in the autocomplete would only lead the admin into an error.
		if u.Email != "" && !alreadyMember[strings.ToLower(u.Email)] {
			emails = append(emails, u.Email)
		}
	}
	slices.Sort(emails)
	if len(emails) > MaxListRows {
		emails = emails[:MaxListRows]
	}
	return emails
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
	memberships := h.config.Service.TenantMembers(r.Context(), h.config.TenantID)
	members := toMemberRows(memberships, h.lookupUser)
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
		UserEmailOptions: h.memberEmailOptions(members),
	}
	h.applyMemberError(r, &d)
	p := h.page("Members", "/members", user, r)
	renderPage(w, r, membersPage(p, d))
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
	role := identitymodel.Role(strings.TrimSpace(r.FormValue("role")))
	if email == "" || role == "" {
		triggerToast(w, "err", "Email and role are required")
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
		[]identitymodel.Role{role},
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

func (h *Handler) doUpdateRole(
	w http.ResponseWriter, r *http.Request,
	tenantID identitymodel.TenantID, actor identitymodel.ActorID, back string,
) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	role := identitymodel.Role(strings.TrimSpace(r.FormValue("role")))
	if role == "" {
		triggerToast(w, "err", "Role is required")
		redirect(w, r, back)
		return
	}
	if err := h.config.Service.UpdateMemberRoles(r.Context(), actor, tenantID, []identitymodel.Role{role}); err != nil {
		triggerToast(w, "err", "Role update failed: "+actionErrorMessage(err))
	} else {
		triggerToast(w, "ok", "Role updated")
	}
	redirect(w, r, back)
}
