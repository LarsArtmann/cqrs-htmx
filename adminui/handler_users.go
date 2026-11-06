package adminui

import (
	"encoding/base64"
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func (h *Handler) usersIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	filter := parseUserFilter(r.URL.Query().Get("filter"))
	matched := filterUsers(h.config.Service.ReadModel().AllUsers(), q, filter)
	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), len(matched), listPageSize)
	d := usersListData{
		Users: matched[offset : offset+limit], Total: len(matched), Search: q,
		Filter: filter, BasePath: h.config.BasePath, Page: page, TotalPages: totalPages,
	}

	if cqrshtmx.RenderPartial(r) {
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
)

// parseUserFilter maps the query param to a chip, refusing unknown values
// so a typo cannot silently mean "no filter".
func parseUserFilter(raw string) userFilter {
	switch userFilter(raw) {
	case userFilterVerified, userFilterUnverified, userFilterMFA, userFilterExternal:
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
	case userFilterNone:
		return true
	default:
		return true
	}
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

	roles := map[string][]identitymodel.Role{}
	if authz := h.config.Service.Authz(); authz != nil {
		if domains, derr := authz.DomainsForUser(shown.ID); derr == nil {
			for _, dom := range domains {
				if rs, rerr := authz.RolesForUser(shown.ID, dom); rerr == nil && len(rs) > 0 {
					roles[dom.Get()] = rs
				}
			}
		}
	}

	p := h.page(shown.Email, "/users", user, r)
	renderPage(w, r, userDetailPage(p, userDetailData{
		User: shown, BasePath: h.config.BasePath, TenantRoles: roles,
		PlatformRoles:        roles[identitymodel.WildcardDomain],
		GrantableAdminRoles:  identitymodel.AdminRoles(),
		IsSelf:               user != nil && user.ID == shown.ID,
		Timeline:             h.userTimeline(shown),
		ConfiguredProviders: h.config.Service.ConfiguredOAuth2Providers(),
		UnlinkExternalBase:  h.config.BasePath + "/users/" + shown.ID.Get().String() + "/external",
	}))
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
	if err := h.config.Service.ChangeDisplayName(r.Context(), target, strings.TrimSpace(r.FormValue("display_name"))); err != nil {
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
	roles := make([]usermgmt.Role, 0, len(checked))
	for _, raw := range checked {
		roles = append(roles, usermgmt.Role(raw))
	}
	if err := h.config.Service.GrantRoles(
		r.Context(),
		usermgmt.ActorIDFromUser(target),
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
		usermgmt.ActorIDFromUser(target),
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
		return identitymodel.UserID{}, err
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
		return userTimeline{}
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
