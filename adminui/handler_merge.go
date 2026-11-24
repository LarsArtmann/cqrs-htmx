package adminui

import (
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// userDuplicates renders the duplicate-users report: every cluster of live
// rows a human would call one person, each row with its merge form.
func (h *Handler) userDuplicates(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	groups := h.config.Service.DuplicateUsers(r.Context())
	rendered := make([]userDuplicateGroup, 0, len(groups))
	for _, group := range groups {
		rows := make([]duplicateUserRow, 0, len(group.Users))
		for _, u := range group.Users {
			survivors := make([]*identitymodel.User, 0, len(group.Users)-1)
			for _, other := range group.Users {
				if other.ID.Get().String() != u.ID.Get().String() {
					survivors = append(survivors, other)
				}
			}
			rows = append(rows, duplicateUserRow{User: u, Survivor: survivors})
		}
		rendered = append(rendered, userDuplicateGroup{
			Key: group.Key, Account: group.Account, Rows: rows,
		})
	}
	d := userDuplicatesData{Groups: rendered, BasePath: h.config.BasePath}
	p := h.page("Duplicate users", "/users", user, r)
	renderPage(w, r, userDuplicatesPage(p, d))
}

// userMerge merges the PATH user into the form field survivor (the route
// contract the duplicates page posts to).
func (h *Handler) userMerge(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	loserID := identitymodel.NewUserID(r.PathValue("id"))
	survivorRaw := strings.TrimSpace(r.FormValue("survivor"))
	if survivorRaw == "" {
		triggerToast(w, "err", "Pick a survivor for the merge")
		h.writeErrorPage(w, r, http.StatusBadRequest, "Survivor required", "Choose which user keeps the identity.")
		return
	}
	survivorID := identitymodel.NewUserID(survivorRaw)
	if err := h.config.Service.MergeUsers(
		r.Context(), survivorID, []identitymodel.UserID{loserID}, "merged via admin panel",
	); err != nil {
		h.writeActionError(w, r, "Merge users", err)
		return
	}
	triggerToast(w, "ok", "Users merged")
	redirect(w, r, h.config.BasePath+"/users/"+survivorID.Get().String())
}

// tenantDuplicates renders the duplicate-tenants report with rename and
// merge forms per row.
func (h *Handler) tenantDuplicates(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	groups := h.config.Service.DuplicateTenants(r.Context())
	d := tenantDuplicatesData{Groups: groups, BasePath: h.config.BasePath}
	p := h.page("Duplicate tenants", "/tenants", user, r)
	renderPage(w, r, tenantDuplicatesPage(p, d))
}

// tenantRename changes a tenant's display name in place (the typo fix that
// does not touch the ID or any tenant-scoped key).
func (h *Handler) tenantRename(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	tenantID := identitymodel.NewTenantID(r.PathValue("id"))
	displayName := strings.TrimSpace(r.FormValue("display_name"))
	if displayName == "" {
		triggerToast(w, "err", "Display name is required")
		h.writeErrorPage(w, r, http.StatusBadRequest, "Name required", "A display name is required.")
		return
	}
	if err := h.config.Service.RenameTenant(r.Context(), tenantID, displayName); err != nil {
		h.writeActionError(w, r, "Rename tenant", err)
		return
	}
	triggerToast(w, "ok", "Tenant renamed")
	redirect(w, r, h.config.BasePath+"/tenants/"+tenantID.Get())
}

// tenantMerge merges the PATH tenant into the form field survivor.
func (h *Handler) tenantMerge(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	loserID := identitymodel.NewTenantID(r.PathValue("id"))
	survivorRaw := strings.TrimSpace(r.FormValue("survivor"))
	if survivorRaw == "" {
		triggerToast(w, "err", "Pick a survivor for the merge")
		h.writeErrorPage(w, r, http.StatusBadRequest, "Survivor required", "Choose which tenant keeps the identity.")
		return
	}
	survivorID := identitymodel.NewTenantID(survivorRaw)
	if err := h.config.Service.MergeTenants(
		r.Context(), survivorID, []identitymodel.TenantID{loserID}, "merged via admin panel",
	); err != nil {
		h.writeActionError(w, r, "Merge tenants", err)
		return
	}
	triggerToast(w, "ok", "Tenants merged")
	redirect(w, r, h.config.BasePath+"/tenants/"+survivorID.Get())
}

// superAdminMergeRoutes are the duplicate-reconciliation surfaces: the two
// duplicates reports and the merge/rename writes they post to. Merging
// identities consolidates authority and history, so every write stays
// super-admin.
//
//nolint:exhaustruct // rows intentionally omit inapplicable fields (asset vs guarded, write)
func (h *Handler) superAdminMergeRoutes() []routeSpec {
	return []routeSpec{
		{
			method:  http.MethodGet,
			pattern: "/users/duplicates",
			panel:   panelUsers,
			guarded: h.userDuplicates,
			mode:    routeModeSuperAdmin,
			why:     "the duplicates report spans every tenant's users",
		},
		{
			method:  http.MethodPost,
			pattern: "/users/{id}/merge",
			panel:   panelUsers,
			guarded: h.userMerge,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "consolidating duplicate identities is terminal and platform-wide",
		},
		{
			method:  http.MethodGet,
			pattern: "/tenants/duplicates",
			panel:   panelTenants,
			guarded: h.tenantDuplicates,
			mode:    routeModeSuperAdmin,
			why:     "the duplicates report is the platform roster's mirror",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/rename",
			panel:   panelTenants,
			guarded: h.tenantRename,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "renaming a tenant relabels every tenant-scoped surface",
		},
		{
			method:  http.MethodPost,
			pattern: "/tenants/{id}/merge",
			panel:   panelTenants,
			guarded: h.tenantMerge,
			mode:    routeModeSuperAdmin,
			write:   true,
			why:     "consolidating duplicate tenants moves every member's authority",
		},
	}
}

