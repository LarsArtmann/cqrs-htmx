package adminui

import (
	"context"
	"encoding/base64"
	"net/http"
	"slices"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	errorfamily "github.com/larsartmann/go-error-family"
)

func (h *Handler) usersIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	filter := parseUserFilter(r.URL.Query().Get("filter"))
	matched := filterUsers(h.config.Service.ReadModel().AllUsers(), q, filter)
	if filter == userFilterAdmin {
		matched = h.filterPlatformAdmins(matched)
	}
	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), len(matched), listPageSize)
	visible := matched[offset : offset+limit]
	rows := make([]userRow, 0, len(visible))
	for _, u := range visible {
		rows = append(rows, h.userRow(r.Context(), u))
	}
	d := usersListData{
		Users: rows, Total: len(matched), Search: q,
		Filter: filter, BasePath: h.config.BasePath, Page: page, TotalPages: totalPages,
	}

	if htmx.RenderPartial(r) {
		renderPartial(w, r, usersTableContent(d))
		return
	}
	p := h.page("Users", "/users", user, r)
	renderPage(w, r, usersPage(p, d))
}

// userFilter is a one-of status chip narrowing the users list. The zero
// value means no filter (all users).
type userFilter string

const (
	userFilterNone       userFilter = ""
	userFilterVerified   userFilter = "verified"
	userFilterUnverified userFilter = "unverified"
	userFilterMFA        userFilter = "mfa"
	userFilterExternal   userFilter = "external"
	userFilterAdmin      userFilter = "admin"
)

// parseUserFilter maps the query param to a chip, refusing unknown values
// so a typo cannot silently mean "no filter".
func parseUserFilter(raw string) userFilter {
	switch userFilter(raw) {
	case userFilterVerified, userFilterUnverified, userFilterMFA, userFilterExternal, userFilterAdmin:
		return userFilter(raw)
	case userFilterNone:
		return userFilterNone
	default:
		return userFilterNone
	}
}

// filterUsers returns users whose email or display name contains q
// (case-insensitive) and that satisfy the status chip. Empty q and no chip
// return all users.
func filterUsers(all []*identitymodel.User, q string, filter userFilter) []*identitymodel.User {
	needle := strings.ToLower(q)
	out := make([]*identitymodel.User, 0, len(all))
	for _, u := range all {
		if needle != "" &&
			!strings.Contains(strings.ToLower(u.Email), needle) &&
			!strings.Contains(strings.ToLower(u.DisplayName), needle) {
			continue
		}
		if !userMatchesFilter(u, filter) {
			continue
		}
		out = append(out, u)
	}
	return out
}

func userMatchesFilter(u *identitymodel.User, filter userFilter) bool {
	switch filter {
	case userFilterVerified:
		return u.EmailVerified
	case userFilterUnverified:
		return !u.EmailVerified
	case userFilterMFA:
		return u.TOTPEnabled
	case userFilterExternal:
		return len(u.ExternalAccounts) > 0
	case userFilterAdmin:
		// Platform roles are resolved per row (they live in the policy engine,
		// not the directory), so this chip narrows AFTER the directory scan —
		// see filterPlatformAdmins.
		return true
	case userFilterNone:
		return true
	default:
		return true
	}
}

// userRow resolves the access facts for one directory entry: the platform
// (wildcard-domain) roles they hold and how many tenants they belong to. A
// policy-engine failure degrades the column to empty rather than failing the
// page — the directory itself is still the truth about who exists.
//
// The tenant count is derived from the domains the policy engine reports AND
// that resolve to a real tenant. Registration grants a user's default roles in
// a domain named after their own stream, so a raw domain count would report a
// phantom "tenant" for every user; the tenant lookup is what makes the number
// mean what it says.
func (h *Handler) userRow(ctx context.Context, u *identitymodel.User) userRow {
	row := userRow{User: u}
	authz := h.config.Service.Authz()
	if authz == nil {
		return row
	}
	if roles, err := authz.RolesForUser(u.ID, identitymodel.NewTenantID(identitymodel.WildcardDomain)); err == nil {
		row.PlatformRoles = roles
	}
	if domains, err := authz.DomainsForUser(u.ID); err == nil {
		for _, dom := range domains {
			if dom.Get() == identitymodel.WildcardDomain {
				continue
			}
			if _, err := h.config.Service.GetTenant(ctx, dom); err == nil {
				row.TenantCount++
			}
		}
	}
	return row
}

// filterPlatformAdmins narrows the directory to users holding a platform
// (wildcard-domain) role. It runs after the directory scan because the roles
// live in the policy engine, not the read model.
func (h *Handler) filterPlatformAdmins(all []*identitymodel.User) []*identitymodel.User {
	out := make([]*identitymodel.User, 0, len(all))
	for _, u := range all {
		if h.userRow(context.Background(), u).isPlatformAdmin() {
			out = append(out, u)
		}
	}
	return out
}

func (h *Handler) userDetail(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	target, err := identitymodel.ParseUserID(r.PathValue("id"))
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid user id",
			"The id in the URL path is not a valid user id.",
		)
		return
	}
	shown, err := h.config.Service.GetUser(r.Context(), target)
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusNotFound,
			"User not found",
			"No user with this id exists (it may have been deleted).",
		)
		return
	}

	platformRoles, tenantAccess := h.userAccess(r.Context(), shown.ID)
	userBase := h.config.BasePath + "/users/" + shown.ID.Get().String()

	p := h.page(shown.Email, "/users", user, r)
	renderPage(w, r, userDetailPage(p, userDetailData{
		User:                shown,
		BasePath:            h.config.BasePath,
		PlatformRoles:       platformRoles,
		GrantableAdminRoles: identitymodel.AdminRoles(),
		IsSelf:              user != nil && user.ID == shown.ID,
		Timeline:            h.userTimeline(shown),
		ConfiguredProviders: h.config.Service.ConfiguredOAuth2Providers(),
		UnlinkExternalBase:  userBase + "/external",
		TenantAccess:        tenantAccess,
		TenantOptions:       withoutTenantOptions(tenantSelectOptions(h.config.Service.AllTenants()), tenantAccess),
		AssignableRoles:     identitymodel.AssignableRoles(),
		AddTenantAccessURL:  userBase + "/tenants",
		UpdateTenantBase:    userBase + "/tenants",
		RemoveTenantBase:    userBase + "/tenants",
	}))
}

// userAccess resolves a user's authority into the two views the detail page
// needs: platform (wildcard-domain) roles and their resolved tenant
// memberships, ordered by tenant name. Domains that no longer resolve to a
// tenant are dropped — registration grants default roles in a domain named
// after the user's own stream, and that is not a tenant.
func (h *Handler) userAccess(
	ctx context.Context, uid identitymodel.UserID,
) (platformRoles []identitymodel.Role, access []tenantAccessRow) {
	authz := h.config.Service.Authz()
	if authz == nil {
		return nil, nil
	}
	if roles, err := authz.RolesForUser(uid, identitymodel.NewTenantID(identitymodel.WildcardDomain)); err == nil {
		platformRoles = roles
	}
	domains, err := authz.DomainsForUser(uid)
	if err != nil {
		return platformRoles, nil
	}
	access = make([]tenantAccessRow, 0, len(domains))
	for _, dom := range domains {
		if dom.Get() == identitymodel.WildcardDomain {
			continue
		}
		roles, rerr := authz.RolesForUser(uid, dom)
		if rerr != nil || len(roles) == 0 {
			continue
		}
		tenant, terr := h.config.Service.GetTenant(ctx, dom)
		if terr != nil {
			continue
		}
		access = append(access, tenantAccessRow{
			TenantID: dom.Get(), Name: tenant.DisplayName, Slug: tenant.Name, Roles: roles,
		})
	}
	slices.SortFunc(access, func(a, b tenantAccessRow) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return platformRoles, access
}

// userGrantTenantAccess adds (or replaces) a membership from the user page:
// dispatches the membership command through the durable event-sourced path, so
// the grant survives restarts and replays.
func (h *Handler) userGrantTenantAccess(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, back, ok := h.parseTargetUser(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	tenantRaw := strings.TrimSpace(r.FormValue("tenant"))
	role := identitymodel.Role(strings.TrimSpace(r.FormValue("role")))
	if tenantRaw == "" || role == "" {
		triggerToast(w, "err", "Pick a tenant and a role")
		redirect(w, r, back)
		return
	}
	tenantID := identitymodel.NewTenantID(tenantRaw)
	if _, err := h.config.Service.GetTenant(r.Context(), tenantID); err != nil {
		triggerToast(w, "err", "Unknown tenant")
		redirect(w, r, back)
		return
	}
	if !slices.Contains(identitymodel.AssignableRoles(), role) {
		triggerToast(w, "err", "Unknown role")
		redirect(w, r, back)
		return
	}
	if err := h.config.Service.AddMember(
		r.Context(), identitymodel.ActorIDFromUser(target), tenantID, []identitymodel.Role{role},
	); err != nil {
		h.writeActionError(w, r, "Grant tenant access", err)
		return
	}
	triggerToast(w, "ok", "Tenant access granted")
	redirect(w, r, back)
}

// userUpdateTenantRole replaces a user's roles in one tenant. It is the
// counterpart of the grant: a membership is ADDED once (the decider refuses a
// duplicate), so changing an existing membership is its own command.
func (h *Handler) userUpdateTenantRole(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, back, ok := h.parseTargetUser(w, r)
	if !ok {
		return
	}
	tenantRaw := strings.TrimSpace(r.PathValue("tenant"))
	if tenantRaw == "" {
		triggerToast(w, "err", "Missing tenant")
		redirect(w, r, back)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	role := identitymodel.Role(strings.TrimSpace(r.FormValue("role")))
	if !slices.Contains(identitymodel.AssignableRoles(), role) {
		triggerToast(w, "err", "Unknown role")
		redirect(w, r, back)
		return
	}
	if err := h.config.Service.UpdateMemberRoles(
		r.Context(), identitymodel.ActorIDFromUser(target), identitymodel.NewTenantID(tenantRaw),
		[]identitymodel.Role{role},
	); err != nil {
		h.writeActionError(w, r, "Update tenant role", err)
		return
	}
	triggerToast(w, "ok", "Tenant role updated")
	redirect(w, r, back)
}

// userRevokeTenantAccess removes one membership from the user page, dispatching
// the same durable RemoveMember command the tenant roster uses.
func (h *Handler) userRevokeTenantAccess(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, back, ok := h.parseTargetUser(w, r)
	if !ok {
		return
	}
	tenantRaw := strings.TrimSpace(r.PathValue("tenant"))
	if tenantRaw == "" {
		triggerToast(w, "err", "Missing tenant")
		redirect(w, r, back)
		return
	}
	tenantID := identitymodel.NewTenantID(tenantRaw)
	if err := h.config.Service.RemoveMember(
		r.Context(), identitymodel.ActorIDFromUser(target), tenantID,
	); err != nil {
		h.writeActionError(w, r, "Revoke tenant access", err)
		return
	}
	triggerToast(w, "ok", "Tenant access revoked")
	redirect(w, r, back)
}

// parseTargetUser parses the {id} path value into a user id and returns the
// user's detail URL as the safe return path. A malformed id answers 400.
func (h *Handler) parseTargetUser(w http.ResponseWriter, r *http.Request) (identitymodel.UserID, string, bool) {
	target, err := identitymodel.ParseUserID(r.PathValue("id"))
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid user id",
			"The id in the URL path is not a valid user id.",
		)
		return identitymodel.UserID{}, "", false
	}
	return target, h.config.BasePath + "/users/" + target.Get().String(), true
}

func (h *Handler) userDelete(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, err := identitymodel.ParseUserID(r.PathValue("id"))
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid user id",
			"The id in the URL path is not a valid user id.",
		)
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		reason = "deleted via admin panel"
	}
	if err := h.config.Service.DeleteUser(r.Context(), target, reason); err != nil {
		h.writeActionError(w, r, "Delete user", err)
		return
	}
	triggerToast(w, "ok", "User deleted")
	redirect(w, r, h.config.BasePath+"/users")
}

// userUnlinkExternal removes a single OAuth2/OIDC provider link from a user.
// It calls the Service's public UnlinkExternalAccount, which enforces the
// last-auth-method guard (rejecting unlink if the user would be left with no
// WebAuthn credentials and no other external accounts).
//
// Linking a provider cannot be initiated from the admin panel: the OAuth2
// handshake requires the user to authenticate with the provider themselves,
// which the admin cannot impersonate. The user-detail card documents this and
// lists the configured providers so the admin knows what the user CAN link.
func (h *Handler) userUnlinkExternal(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, err := identitymodel.ParseUserID(r.PathValue("id"))
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid user id",
			"The id in the URL path is not a valid user id.",
		)
		return
	}
	provider := strings.TrimSpace(r.PathValue("provider"))
	if provider == "" {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Missing provider", "The provider path segment is required.")
		return
	}
	if err := h.config.Service.UnlinkExternalAccount(r.Context(), target, provider); err != nil {
		h.writeActionError(w, r, "Unlink "+provider, err)
		return
	}
	triggerToast(w, "ok", provider+" account unlinked")
	redirect(w, r, h.config.BasePath+"/users/"+target.Get().String())
}

func (h *Handler) userNew(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("New user", "/users", user, r)
	renderPage(w, r, userNewPage(p, h.config.BasePath))
}

// userCreate registers a user through the Service's public Register path.
// Register also mints a session for the new user; the panel discards it —
// it is never exposed to the admin's browser — and the user still signs in
// themselves via their own passkey or OAuth2 flow (the panel cannot set
// credentials; the form documents this).
func (h *Handler) userCreate(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	req := usermgmt.RegisterRequest{
		Email:       strings.TrimSpace(r.FormValue("email")),
		DisplayName: strings.TrimSpace(r.FormValue("display_name")),
	}
	if req.Email == "" {
		triggerToast(w, "err", "Email is required")
		h.writeErrorPage(w, r, http.StatusBadRequest, "Email required", "An email address is required.")
		return
	}
	result, err := h.config.Service.Register(r.Context(), req)
	if err != nil {
		h.writeActionError(w, r, "Create user", err)
		return
	}
	triggerToast(w, "ok", "User created")
	redirect(w, r, h.config.BasePath+"/users/"+result.User.ID.Get().String())
}

func (h *Handler) userUpdateDisplayName(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, err := h.parseUserID(w, r)
	if err != nil {
		return
	}
	if err := h.config.Service.ChangeDisplayName(
		r.Context(),
		target,
		strings.TrimSpace(r.FormValue("display_name")),
	); err != nil {
		h.writeActionError(w, r, "Change display name", err)
		return
	}
	triggerToast(w, "ok", "Display name updated")
	redirect(w, r, h.userPath(target))
}

func (h *Handler) userUpdateEmail(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, err := h.parseUserID(w, r)
	if err != nil {
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		triggerToast(w, "err", "Email is required")
		h.writeErrorPage(w, r, http.StatusBadRequest, "Email required", "An email address is required.")
		return
	}
	if err := h.config.Service.ChangeEmail(r.Context(), target, email); err != nil {
		h.writeActionError(w, r, "Change email", err)
		return
	}
	triggerToast(w, "ok", "Email updated")
	redirect(w, r, h.userPath(target))
}

// userGrantPlatformRoles grants exactly the checked roles in the wildcard
// domain via the durable GrantRoles path (ADR-0057): the grant becomes
// membership events, so it survives restarts and projection rebuilds.
func (h *Handler) userGrantPlatformRoles(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, err := h.parseUserID(w, r)
	if err != nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	checked := r.Form["roles"]
	if len(checked) == 0 {
		h.writeActionError(w, r, "Grant roles", errConfig("select at least one platform role"))
		return
	}
	roles := make([]identitymodel.Role, 0, len(checked))
	for _, raw := range checked {
		roles = append(roles, identitymodel.Role(raw))
	}
	if err := h.config.Service.GrantRoles(
		r.Context(),
		identitymodel.ActorIDFromUser(target),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		roles,
	); err != nil {
		h.writeActionError(w, r, "Grant roles", err)
		return
	}
	triggerToast(w, "ok", "Platform roles granted")
	redirect(w, r, h.userPath(target))
}

// userRevokePlatformRoles removes the user's wildcard-domain membership, so
// they hold no platform roles at all. Revoking your own admin grant locks you
// out of the panel — the confirm dialog says so.
func (h *Handler) userRevokePlatformRoles(w http.ResponseWriter, r *http.Request, session *identitymodel.User) {
	target, err := h.parseUserID(w, r)
	if err != nil {
		return
	}
	if err := h.config.Service.RevokeRoles(
		r.Context(),
		identitymodel.ActorIDFromUser(target),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
	); err != nil {
		h.writeActionError(w, r, "Revoke roles", err)
		return
	}
	if session != nil && session.ID == target {
		triggerToast(w, "ok", "Platform roles revoked — you may have locked yourself out")
	} else {
		triggerToast(w, "ok", "Platform roles revoked")
	}
	redirect(w, r, h.userPath(target))
}

// userRemoveCredential detaches one passkey. The credential id travels
// base64url-encoded in the path because raw WebAuthn ids are binary.
func (h *Handler) userRemoveCredential(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, err := h.parseUserID(w, r)
	if err != nil {
		return
	}
	credentialID, err := base64.RawURLEncoding.DecodeString(r.PathValue("credential"))
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid credential id",
			"The credential id in the URL path is not valid base64url.",
		)
		return
	}
	if err := h.config.Service.RemoveCredential(r.Context(), target, credentialID); err != nil {
		h.writeActionError(w, r, "Remove passkey", err)
		return
	}
	triggerToast(w, "ok", "Passkey removed")
	redirect(w, r, h.userPath(target))
}

// userResendVerification re-issues the email-verification token. When the
// Service has no verification configured the action error explains it instead
// of surfacing an opaque 500.
func (h *Handler) userResendVerification(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	target, err := h.parseUserID(w, r)
	if err != nil {
		return
	}
	if _, err := h.config.Service.SendVerificationEmail(r.Context(), target); err != nil {
		h.writeActionError(w, r, "Resend verification", err)
		return
	}
	triggerToast(w, "ok", "Verification email sent")
	redirect(w, r, h.userPath(target))
}

// parseUserID parses the {id} path value, writing the error page itself when
// it is invalid. The bool return reports whether the caller may continue.
func (h *Handler) parseUserID(w http.ResponseWriter, r *http.Request) (identitymodel.UserID, error) {
	target, err := identitymodel.ParseUserID(r.PathValue("id"))
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid user id",
			"The id in the URL path is not a valid user id.",
		)
		return identitymodel.UserID{}, errorfamily.WrapRejection(
			err,
			"adminui.invalid_user_id",
			"the id in the URL path is not a valid user id",
		)
	}
	return target, nil
}

func (h *Handler) userPath(id identitymodel.UserID) string {
	return h.config.BasePath + "/users/" + id.Get().String()
}

// userTimeline collects the audit trail for ONE user's aggregate stream —
// events done BY them and done TO them (admin role grants, email changes,
// deletions) alike, because both live on the user's stream. The AuditLog is
// keyed by StreamID, and a user's stream id is the same ULID as their user
// id, so matching on the string form avoids importing the id package. The
// full audit page stays the deep-dive; this is the per-user glance.
func (h *Handler) userTimeline(shown *identitymodel.User) userTimeline {
	al := h.config.Service.AuditLog()
	if al == nil {
		return userTimeline{ // the zero timeline: no entries, no total
			Entries: nil,
			Total:   0,
		}
	}
	want := shown.ID.Get().String()
	all := al.Recent(al.Count())
	entries := make([]usermgmt.AuditEntry, 0, 16)
	for _, e := range all {
		if e.AggregateID.Get() == want {
			entries = append(entries, e)
			if len(entries) >= 8 {
				break
			}
		}
	}
	resolveAuditEmails(h.config.Service, entries)
	total := 0
	for _, e := range all {
		if e.AggregateID.Get() == want {
			total++
		}
	}
	return userTimeline{Entries: entries, Total: total}
}
