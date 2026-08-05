package adminui

import (
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func (h *Handler) usersIndex(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	matched := filterUsers(h.config.Service.ReadModel().AllUsers(), q)
	users, total := capList(matched)
	d := usersListData{Users: users, Total: total, Search: q, BasePath: h.config.BasePath}

	if cqrshtmx.RenderPartial(r) {
		renderPartial(w, r, usersTableContent(d))
		return
	}
	p := h.page("Users", "/users", user, r)
	renderPage(w, r, usersPage(p, d))
}

// filterUsers returns users whose email or display name contains q
// (case-insensitive). Empty q returns all users.
func filterUsers(all []*usermgmt.User, q string) []*usermgmt.User {
	if q == "" {
		return all
	}
	needle := strings.ToLower(q)
	out := make([]*usermgmt.User, 0, len(all))
	for _, u := range all {
		if strings.Contains(strings.ToLower(u.Email), needle) ||
			strings.Contains(strings.ToLower(u.DisplayName), needle) {
			out = append(out, u)
		}
	}
	return out
}

func (h *Handler) userDetail(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	target, err := usermgmt.ParseUserID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	shown, err := h.config.Service.GetUser(r.Context(), target)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	roles := map[string][]usermgmt.Role{}
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

func (h *Handler) userDelete(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	target, err := usermgmt.ParseUserID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		reason = "deleted via admin panel"
	}
	if err := h.config.Service.DeleteUser(r.Context(), target, reason); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		w.WriteHeader(http.StatusBadRequest)
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
func (h *Handler) userUnlinkExternal(w http.ResponseWriter, r *http.Request, _ *usermgmt.User) {
	target, err := usermgmt.ParseUserID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	provider := strings.TrimSpace(r.PathValue("provider"))
	if provider == "" {
		http.Error(w, "missing provider", http.StatusBadRequest)
		return
	}
	if err := h.config.Service.UnlinkExternalAccount(r.Context(), target, provider); err != nil {
		triggerToast(w, "err", "Unlink failed: "+err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", provider+" account unlinked")
	redirect(w, r, h.config.BasePath+"/users/"+target.Get().String())
}
