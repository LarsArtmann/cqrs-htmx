package adminui

import (
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func (h *Handler) usersIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	matched := filterUsers(h.config.Service.ReadModel().AllUsers(), q)
	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), len(matched), listPageSize)
	d := usersListData{
		Users: matched[offset : offset+limit], Total: len(matched), Search: q,
		BasePath: h.config.BasePath, listPage: listPage{Page: page, TotalPages: totalPages},
	}

	if cqrshtmx.RenderPartial(r) {
		renderPartial(w, r, usersTableContent(d))
		return
	}
	p := h.page("Users", "/users", user, r)
	renderPage(w, r, usersPage(p, d))
}

// filterUsers returns users whose email or display name contains q
// (case-insensitive). Empty q returns all users.
func filterUsers(all []*identitymodel.User, q string) []*identitymodel.User {
	if q == "" {
		return all
	}
	needle := strings.ToLower(q)
	out := make([]*identitymodel.User, 0, len(all))
	for _, u := range all {
		if strings.Contains(strings.ToLower(u.Email), needle) ||
			strings.Contains(strings.ToLower(u.DisplayName), needle) {
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
