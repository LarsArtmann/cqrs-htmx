package adminui

import (
	"context"
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

func (h *Handler) tenantsIndex(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := parseTenantStatusFilter(r.URL.Query().Get("status"))
	sort := parseTenantSort(r.URL.Query().Get("sort"))
	matched := make([]*usermgmt.Tenant, 0)
	for _, t := range h.config.Service.AllTenants() {
		if !matchesTenantSearch(t, q) || !matchesTenantStatus(t, status) {
			continue
		}
		matched = append(matched, t)
	}
	// Member counts: the members sort ranks every MATCHED tenant (the order
	// spans pages), so only that ordering pays for the full map; any other
	// order reads counts for the visible page's rows alone.
	if sort == "members" {
		sortTenants(matched, sort, tenantMemberCounts(r.Context(), h.config.Service, matched))
	}
	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), len(matched), h.config.PageSize)
	visible := matched[offset : offset+limit]
	counts := tenantMemberCounts(r.Context(), h.config.Service, visible)
	rows := make([]tenantRow, 0, limit)
	for _, t := range visible {
		rows = append(rows, tenantRow{
			Tenant:      t,
			MemberCount: counts[t.ID],
		})
	}
	d := tenantsListData{
		Tenants: rows, Total: len(matched), Search: q,
		Status: status, Sort: sort, StatusCounts: tenantStatusCounts(matched),
		BasePath: h.config.BasePath,
		Page:     page, TotalPages: totalPages,
	}
	p := h.page("Tenants", "/tenants", user, r)
	if htmx.RenderPartial(r) {
		renderPartial(w, r, tenantsTableContent(p, d))
		return
	}
	renderPage(w, r, tenantsPage(p, d))
}

// tenantMemberCounts resolves one TenantMembers query per tenant in tenants
// (the read model has no batched count query yet), so callers keep tenants
// bounded to the rows they will actually render — except the members sort,
// which legitimately spans pages.
func tenantMemberCounts(
	ctx context.Context, svc *usermgmt.Service, tenants []*usermgmt.Tenant,
) map[identitymodel.TenantID]int {
	counts := make(map[identitymodel.TenantID]int, len(tenants))
	for _, t := range tenants {
		counts[t.ID] = len(svc.TenantMembers(ctx, t.ID))
	}
	return counts
}

func (h *Handler) tenantNew(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	p := h.page("New tenant", "/tenants", user, r)
	renderPage(w, r, tenantNewPage(p, h.config.BasePath))
}

func (h *Handler) tenantCreate(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	display := strings.TrimSpace(r.FormValue("display_name"))
	if display == "" {
		display = name
	}
	if name == "" {
		triggerToast(w, "err", "Tenant name is required")
		h.writeErrorPage(w, r, http.StatusBadRequest, "Name required", "A tenant name is required.")
		return
	}
	tenant, err := h.config.Service.CreateTenant(r.Context(), usermgmt.CreateTenantRequest{
		ID:          identitymodel.NewTenantID(name),
		Name:        name,
		DisplayName: display,
	})
	if err != nil {
		h.writeActionError(w, r, "Create tenant", err)
		return
	}
	triggerToast(w, "ok", "Tenant created")
	redirect(w, r, h.config.BasePath+"/tenants/"+tenant.ID.Get())
}

func (h *Handler) tenantDetail(w http.ResponseWriter, r *http.Request, user *identitymodel.User) {
	tenantID := identitymodel.NewTenantID(r.PathValue("id"))
	tenant, err := h.config.Service.GetTenant(r.Context(), tenantID)
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusNotFound,
			"Tenant not found",
			"No tenant with this id exists (it may have been deleted).",
		)
		return
	}
	memberships := h.config.Service.TenantMembers(r.Context(), tenantID)
	members := toMemberRows(memberships, h.lookupUser)
	addedBy := h.addedByIndex(r.Context(), tenantID, members)
	for i := range members {
		members[i].AddedBy = addedBy[members[i].Actor.PrefixedString()]
	}
	memberBase := h.config.BasePath + "/tenants/" + tenantID.Get() + "/members"
	d := tenantDetailData{
		Tenant:              tenant,
		Members:             members,
		AssignableRoles:     identitymodel.AssignableRoles(),
		BasePath:            h.config.BasePath,
		AddMemberURL:        memberBase,
		RemoveMemberBase:    memberBase,
		UpdateRoleBase:      memberBase,
		UserDetailBase:      h.config.BasePath + "/users/",
		UserEmailOptions:    h.memberEmailOptions(members),
		AddMemberCreateHref: h.config.BasePath + "/users/new",
		// F080: the identity card answers "who holds which roles here" from
		// the rows already resolved — no extra queries.
		RoleDistribution: roleDistribution(members),
	}
	h.applyMemberError(r, &d)
	p := h.page(tenant.DisplayName, "/tenants", user, r)
	// The switcher is rebuilt with the viewed tenant marked, so the dropdown
	// answers "where am I" (T091: tenant detail is the tenant dashboard the
	// switcher falls back to from any other page).
	p.CurrentTenantID = tenantID.Get()
	p.TenantSwitcher = h.tenantSwitchOptions(tenantID.Get())
	renderPage(w, r, tenantDetailPage(p, d))
}

func (h *Handler) tenantSuspend(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.SuspendTenant(r.Context(), id, "suspended via admin panel"); err != nil {
		h.writeActionError(w, r, "Suspend tenant", err)
		return
	}
	triggerToast(w, "ok", "Tenant suspended")
	redirect(w, r, h.config.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantReactivate(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.ReactivateTenant(r.Context(), id); err != nil {
		h.writeActionError(w, r, "Reactivate tenant", err)
		return
	}
	triggerToast(w, "ok", "Tenant reactivated")
	redirect(w, r, h.config.BasePath+"/tenants/"+id.Get())
}

func (h *Handler) tenantDelete(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	id := identitymodel.NewTenantID(r.PathValue("id"))
	if err := h.config.Service.DeleteTenant(r.Context(), id, "deleted via admin panel"); err != nil {
		h.writeActionError(w, r, "Delete tenant", err)
		return
	}
	triggerToast(w, "ok", "Tenant deleted")
	redirect(w, r, h.config.BasePath+"/tenants")
}
