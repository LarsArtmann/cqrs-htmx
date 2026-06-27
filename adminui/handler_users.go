package adminui

import (
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

func (h *Handler) usersIndex(w http.ResponseWriter, r *http.Request, user *usermgmt.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	matched := filterUsers(h.cfg.Service.ReadModel().AllUsers(), q)
	users, total := capList(matched)
	d := usersListData{Users: users, Total: total, Search: q, BasePath: h.cfg.BasePath}

	if r.Header.Get("HX-Request") == "true" {
		renderPartial(w, r, usersTable(d))
		return
	}
	p := h.page("Users", "/users", user)
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
	shown, err := h.cfg.Service.GetUser(r.Context(), target)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	roles := map[string][]usermgmt.Role{}
	if authz := h.cfg.Service.Authz(); authz != nil {
		if domains, derr := authz.DomainsForUser(shown.ID); derr == nil {
			for _, dom := range domains {
				if rs, rerr := authz.RolesForUser(shown.ID, dom); rerr == nil && len(rs) > 0 {
					roles[dom] = rs
				}
			}
		}
	}

	p := h.page(shown.Email, "/users", user)
	renderPage(w, r, userDetailPage(p, userDetailData{
		User: shown, BasePath: h.cfg.BasePath, TenantRoles: roles,
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
	if err := h.cfg.Service.DeleteUser(r.Context(), target, reason); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	triggerToast(w, "ok", "User deleted")
	redirect(w, r, h.cfg.BasePath+"/users")
}
