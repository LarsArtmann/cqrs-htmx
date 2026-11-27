package adminui

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
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
	// spans pages), so it reads the full map once and the rows below reuse it;
	// any other order reads counts for the visible page's rows alone. Either
	// way a tenant's count is read exactly once per request.
	var counts map[identitymodel.TenantID]int
	if sort == "members" {
		counts = h.tenantMemberCounts(r.Context(), matched)
	}

	// EVERY order is applied before paging. The read model returns tenants in
	// map order, so a list that skipped this would page differently on every
	// request (2026-11-27: the members-only guard introduced on 11-24 dropped
	// the default name sort).
	sortTenants(matched, sort, counts)

	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), len(matched), h.config.PageSize)
	visible := matched[offset : offset+limit]
	if sort != "members" {
		counts = h.tenantMemberCounts(r.Context(), visible)
	}
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

// tenantMemberCounts resolves one member-count read per tenant in tenants
// through the Handler's read seam (the read model has no batched count query
// yet), so callers keep tenants bounded to the rows they will actually render
// — except the members sort, which legitimately spans pages.
func (h *Handler) tenantMemberCounts(
	ctx context.Context, tenants []*usermgmt.Tenant,
) map[identitymodel.TenantID]int {
	counts := make(map[identitymodel.TenantID]int, len(tenants))
	for _, t := range tenants {
		counts[t.ID] = h.memberCount(ctx, t.ID)
	}
	return counts
}

// countTenantMembers binds the service read behind Handler.memberCount: one
// TenantMembers query, whose length is the tenant's member count.
func countTenantMembers(svc *usermgmt.Service) func(ctx context.Context, tenantID identitymodel.TenantID) int {
	return func(ctx context.Context, tenantID identitymodel.TenantID) int {
		return len(svc.TenantMembers(ctx, tenantID))
	}
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
		// F155: a DELETED tenant is allowed into the detail route — but only
		// into the tombstone view. A tenant with no deletion entry in the
		// event record is a genuine 404, not a tombstone candidate.
		if tb := h.tenantTombstone(r.Context(), tenantID); tb.Recorded {
			h.renderTenantTombstone(w, r, user, tenantID, tb)
			return
		}
		h.writeErrorPage(
			w,
			r,
			http.StatusNotFound,
			"Tenant not found",
			"No tenant with this id exists (it may have been deleted).",
		)
		return
	}
	// Defensive second lock: a read model that keeps deleted tenants marked
	// still renders ONLY the tombstone here, never the live view below.
	if tenant.Deleted {
		h.renderTenantTombstone(w, r, user, tenantID, h.tenantTombstone(r.Context(), tenantID))
		return
	}
	allMembers := h.tenantRoster(r.Context(), tenantID)
	total := len(allMembers)
	// F157: the roster paginates beyond MaxListRows like every other list —
	// one page of rows, the full count for the captions and confirms.
	offset, limit, totalPages, page := pageBounds(parsePageQuery(r), total, h.config.PageSize)
	members := allMembers[offset : offset+limit]
	addedBy := h.addedByIndex(r.Context(), tenantID, members)
	for i := range members {
		members[i].AddedBy = addedBy[members[i].Actor.PrefixedString()]
	}
	// The page (and its merged label table) is built before the data so the
	// transfer candidates' confirm bodies interpolate through the labels
	// (f#19) — the labels must exist before any confirm text is composed.
	p := h.page(tenant.DisplayName, "/tenants", user, r)
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
		SuggestURL:          h.config.BasePath + "/users/suggest",
		AddMemberCreateHref: h.config.BasePath + "/users/new",
		// F080: the identity card answers "who holds which roles here" from
		// the rows already resolved — no extra queries.
		RoleDistribution:   roleDistribution(allMembers),
		TotalMembers:       total,
		AdminTierCount:     countAdminTier(allMembers),
		Page:               page,
		TotalPages:         totalPages,
		RosterURL:          h.config.BasePath + "/tenants/" + tenantID.Get(),
		RosterExportURL:    h.config.BasePath + "/tenants/" + tenantID.Get() + "/roster.csv",
		TransferURL:        h.config.BasePath + "/tenants/" + tenantID.Get() + "/transfer",
		TransferCandidates: transferCandidates(p.Labels, allMembers),
	}
	h.applyMemberError(r, &d)
	// The switcher is rebuilt with the viewed tenant marked, so the dropdown
	// answers "where am I" (T091: tenant detail is the tenant dashboard the
	// switcher falls back to from any other page).
	p.CurrentTenantID = tenantID.Get()
	p.TenantSwitcher = h.tenantSwitchOptions(tenantID.Get())
	renderPage(w, r, tenantDetailPage(p, d))
}

// renderTenantTombstone renders the read-only tombstone view (F154/F155).
// F154 honesty: the display name lived only in the read model (which drops
// deleted tenants), so the tombstone shows the slug as the identity and
// leaves the display name "not recorded" rather than guessing.
func (h *Handler) renderTenantTombstone(
	w http.ResponseWriter, r *http.Request, user *identitymodel.User,
	tenantID identitymodel.TenantID, tb *tenantTombstone,
) {
	//nolint:exhaustruct // a tombstone ghost carries only identity facts
	ghost := &usermgmt.Tenant{
		ID: tenantID, Name: tenantID.Get(), DisplayName: tenantID.Get(), Deleted: true,
	}
	d := tenantDetailData{
		Tenant:    ghost,
		BasePath:  h.config.BasePath,
		Tombstone: tb,
	}
	p := h.page(ghost.DisplayName, "/tenants", user, r)
	renderPage(w, r, tenantTombstonePage(p, d))
}

// tenantRoster assembles the tenant's full member rows (the person behind
// each membership resolved through the directory). Both the detail page's
// paginated view and the CSV export read through it, so the page and the
// export can never disagree about who is a member.
func (h *Handler) tenantRoster(ctx context.Context, tenantID identitymodel.TenantID) []memberRow {
	return toMemberRows(h.config.Service.TenantMembers(ctx, tenantID), h.lookupUser)
}

// countAdminTier counts admin/owner holders across a member slice.
func countAdminTier(members []memberRow) int {
	count := 0
	for _, m := range members {
		if holdsAdminTier(m.Roles) {
			count++
		}
	}
	return count
}

// transferCandidates lists the members eligible to RECEIVE ownership: every
// member except the current owner-role holders (transferring to an existing
// owner is refused server-side, so offering them would lead the admin into
// an error). Each candidate carries the two-sided confirm body naming the
// prior owner and the receiver by their directory identity (F163),
// interpolated through the merged label table (f#19).
func transferCandidates(labels map[string]string, members []memberRow) []transferCandidate {
	priorName := ""
	for _, m := range members {
		if hasRole(m.Roles, identitymodel.RoleOwner) {
			priorName = m.displayName()
			break
		}
	}
	candidates := make([]transferCandidate, 0, len(members))
	for _, m := range members {
		if hasRole(m.Roles, identitymodel.RoleOwner) {
			continue
		}
		c := transferCandidate{Email: m.displayName(), ConfirmBody: ""}
		if priorName != "" {
			c.ConfirmBody = tenantTransferConfirm(labels, priorName, m.displayName())
		} else {
			c.ConfirmBody = tenantTransferGrantOnlyConfirm(labels, m.displayName())
		}
		candidates = append(candidates, c)
	}
	return candidates
}

// tenantTombstone resolves a deleted tenant's deletion facts from the event
// record (F154). Honest scope note: the AuditLog projection subscribes only
// to user and auth events (usermgmt allAuditedEventTypes), so TENANT events
// never appear in it — the durable record for a deletion is the tenant
// stream in the journal. The last TenantDeleted event names when the tenant
// died and (via the same actor chain the audit trail rides, go-cqrs-lite's
// ActorEnricher) who dispatched it. Honesty rules: a load failure, a stream
// without a deletion event, or a pre-actor-chain event yields Recorded=false
// or "not recorded" — the view never guesses a time or blames an actor.
func (h *Handler) tenantTombstone(ctx context.Context, tenantID identitymodel.TenantID) *tenantTombstone {
	//nolint:exhaustruct // zero facts until the record proves them
	tb := &tenantTombstone{}
	stream, err := id.ParseStreamID(tenantID.Get())
	if err != nil {
		return tb // unparseable id: honest "not recorded", never a guess
	}
	events, err := h.config.Service.Journal().Load(
		ctx, id.NewStreamRef(identitymodel.AggregateTypeTenant, stream),
	)
	if err != nil {
		return tb
	}
	for _, evt := range events {
		if evt.Type() != identitymodel.EventTenantDeleted {
			continue
		}
		tb.DeletedAt, tb.Recorded = evt.OccurredAt(), true
		actor := evt.Metadata().ActorID
		if uid, ok := identitymodel.ActorIDAsUserID(actor); ok {
			if u, found := h.lookupUser(uid); found && u.Email != "" {
				tb.DeletedBy = u.Email
			}
		}
		if tb.DeletedBy == "" && !actor.IsZero() {
			tb.DeletedBy = actor.PrefixedString()
		}
	}
	return tb
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

// tenantRosterCSV streams the tenant's FULL roster as CSV (F156): every
// member on every page, never just the visible window — a silently truncated
// export would wear a full roster's name. Mounted super-admin only (the
// /tenants/{id} surface it mirrors is super-admin; the tenant-admin roster
// exports via /members/roster.csv with the same writer).
func (h *Handler) tenantRosterCSV(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	tenantID := identitymodel.NewTenantID(r.PathValue("id"))
	if _, err := h.config.Service.GetTenant(r.Context(), tenantID); err != nil {
		h.writeErrorPage(w, r, http.StatusNotFound, "Tenant not found", "No tenant with this id exists.")
		return
	}
	h.writeRosterCSV(w, r, tenantID, "roster-export-"+tenantID.Get()+".csv")
}

// writeRosterCSV is the shared roster export body (F156). The wire shape:
// actor, email, display name, roles, added-at — the facts the roster page
// shows, in the order it shows them.
func (h *Handler) writeRosterCSV(w http.ResponseWriter, r *http.Request, tenantID identitymodel.TenantID, name string) {
	members := h.tenantRoster(r.Context(), tenantID)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"actor", "email", "display_name", "roles", "added_at"}); err != nil {
		triggerToast(w, "err", "Roster export failed: "+actionErrorMessage(err))
		return
	}
	for _, m := range members {
		roles := make([]string, 0, len(m.Roles))
		for _, role := range m.Roles {
			roles = append(roles, string(role))
		}
		added := ""
		if !m.AddedAt.IsZero() {
			added = m.AddedAt.In(h.clock().Location()).Format(time.RFC3339)
		}
		if err := cw.Write([]string{
			m.Actor.PrefixedString(), m.Email, m.Name,
			strings.Join(roles, " "), added,
		}); err != nil {
			// Headers are committed; the partial output is honest about
			// failing rather than silently truncated.
			triggerToast(w, "err", "Roster export failed: "+actionErrorMessage(err))
			return
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		triggerToast(w, "err", "Roster export failed: "+actionErrorMessage(err))
	}
}

// tenantTransferOwnership is the ONE-confirm, TWO-command ownership transfer
// (T22). Sequencing + refusal design: docs/planning/2026-11-24_T22-ownership-transfer-design.md.
func (h *Handler) tenantTransferOwnership(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	back := h.config.BasePath + "/tenants/" + r.PathValue("id")
	tenantID := identitymodel.NewTenantID(r.PathValue("id"))
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	target, ok := h.config.Service.ReadModel().FindByEmail(email)
	if !ok {
		triggerToast(w, "err", "No account matches "+email+". Nothing changed.")
		redirect(w, r, back)
		return
	}
	targetActor := identitymodel.ActorIDFromUser(target.ID)

	members := h.tenantRoster(r.Context(), tenantID)
	targetRow, found := findMember(members, targetActor)
	if refusal := transferRefusal(members, targetActor, email, found, targetRow); refusal != "" {
		triggerToast(w, "err", refusal)
		redirect(w, r, back)
		return
	}

	// Resolve the prior owner from the roster snapshot: the first owner-role
	// member other than the target. transferRefusal guarantees at most one.
	var prior *memberRow
	for i := range members {
		if hasRole(members[i].Roles, identitymodel.RoleOwner) &&
			members[i].Actor.PrefixedString() != targetActor.PrefixedString() {
			prior = &members[i]
			break
		}
	}

	out := executeOwnershipTransfer(r.Context(), h.config.Service, tenantID, targetActor, targetRow.Roles, prior)
	out.TargetEmail = email
	triggerToast(w, toastKindFor(out), ownershipOutcomeMessage(out))
	redirect(w, r, back)
}

// transferRefusal enumerates the pre-dispatch refusal cases (F162): every
// refusal here lands NOTHING — no grant, no demotion, no partial state.
//   - target not a member: the confirm promised a member-to-member handover;
//     creating a membership behind the admin's back would break that promise.
//   - target already owner (covers self-transfer: the current owner naming
//     themselves): the state it asks for already holds.
//   - ambiguous prior: with two other owners the panel cannot pick which to
//     demote, and guessing would silently strip the wrong owner.
func transferRefusal(
	members []memberRow, target identitymodel.ActorID, email string,
	found bool, targetRow memberRow,
) string {
	if !found {
		return email + " is not a member of this tenant. Add them as a member first. Nothing changed."
	}
	if hasRole(targetRow.Roles, identitymodel.RoleOwner) {
		return email + " already owns this tenant. Nothing changed."
	}
	others := 0
	for _, m := range members {
		if hasRole(m.Roles, identitymodel.RoleOwner) &&
			m.Actor.PrefixedString() != target.PrefixedString() {
			others++
		}
	}
	if others > 1 {
		return "This tenant has multiple owners — demote one manually before transferring. Nothing changed."
	}
	return ""
}

func findMember(members []memberRow, actor identitymodel.ActorID) (memberRow, bool) {
	for _, m := range members {
		if m.Actor.PrefixedString() == actor.PrefixedString() {
			return m, true
		}
	}
	return memberRow{}, false
}

// transferOutcome is what actually landed (F164): the handler's toast is
// composed FROM this, never from the happy path.
type transferOutcome struct {
	TargetEmail string
	PriorEmail  string
	// PriorExisted distinguishes "no prior owner to demote" (grant-only, an
	// honest note) from "prior existed and the demote result is below".
	PriorExisted bool
	PriorRemoved bool // prior held no other role: the membership was revoked
	GrantedOK    bool
	GrantErr     error
	DemotedOK    bool
	DemoteErr    error
}

// executeOwnershipTransfer runs the two durable commands in order (F164):
// grant first — a failed demotion leaves TWO owners, the safe direction;
// demoting first could strand the tenant ownerless. svc is a parameter so
// tests can drive mid-sequence failures against a real service.
func executeOwnershipTransfer(
	ctx context.Context, svc *usermgmt.Service, tenantID identitymodel.TenantID,
	target identitymodel.ActorID, targetRoles []identitymodel.Role, prior *memberRow,
) transferOutcome {
	//nolint:exhaustruct // zero-value outcome until each step reports
	out := transferOutcome{PriorExisted: prior != nil}
	if prior != nil {
		out.PriorEmail = prior.Email
	}

	// Grant: the target's EXISTING roles plus owner — GrantRoles replaces the
	// role set, so granting bare [owner] would silently strip their roles.
	if err := svc.GrantRoles(
		ctx,
		target,
		tenantID,
		appendRole(slices.Clone(targetRoles), identitymodel.RoleOwner),
	); err != nil {
		out.GrantErr = err
		return out // nothing else dispatched: the failure is total, not partial
	}
	out.GrantedOK = true

	if prior == nil {
		return out // grant-only: there was nobody to demote
	}

	remaining := withoutRole(prior.Roles, identitymodel.RoleOwner)
	if len(remaining) == 0 {
		// RevokeRoles removes the whole membership (service_grants.go:45);
		// that is the honest demotion when owner was their only role.
		err := svc.RevokeRoles(ctx, prior.Actor, tenantID)
		out.DemotedOK = err == nil
		out.DemoteErr = err
		out.PriorRemoved = err == nil
		return out
	}
	err := svc.UpdateMemberRoles(ctx, prior.Actor, tenantID, remaining)
	out.DemotedOK = err == nil
	out.DemoteErr = err
	return out
}

// toastKindFor picks the toast severity from what landed: a partial state
// (grant landed, demotion failed) must not render as a plain success.
func toastKindFor(out transferOutcome) string {
	if out.GrantErr != nil || out.DemoteErr != nil {
		return "err"
	}
	return "ok"
}

// ownershipOutcomeMessage says EXACTLY what landed (F164) — including the
// mid-sequence case: the grant succeeded, the demotion did not, and the
// message names both sides plus the failure reason.
func ownershipOutcomeMessage(out transferOutcome) string {
	if out.GrantErr != nil {
		return "Ownership transfer failed: " + actionErrorMessage(out.GrantErr) +
			". Nothing changed."
	}
	if out.DemoteErr != nil {
		return "Owner granted to " + out.TargetEmail + ", but demotion of " + out.PriorEmail +
			" failed: " + actionErrorMessage(out.DemoteErr) + ". " + out.PriorEmail +
			" STILL HOLDS the owner role."
	}
	if !out.PriorExisted {
		return "Ownership granted to " + out.TargetEmail +
			". This tenant had no prior owner to demote."
	}
	if out.PriorRemoved {
		return "Ownership transferred to " + out.TargetEmail + ". " + out.PriorEmail +
			" held no other role, so their membership was removed."
	}
	return "Ownership transferred to " + out.TargetEmail + ". " + out.PriorEmail + " demoted."
}

// appendRole adds role unless already present.
func appendRole(roles []identitymodel.Role, role identitymodel.Role) []identitymodel.Role {
	if !hasRole(roles, role) {
		roles = append(roles, role)
	}
	return roles
}

// withoutRole returns roles minus the given role.
func withoutRole(roles []identitymodel.Role, role identitymodel.Role) []identitymodel.Role {
	out := make([]identitymodel.Role, 0, len(roles))
	for _, r := range roles {
		if r != role {
			out = append(out, r)
		}
	}
	return out
}
