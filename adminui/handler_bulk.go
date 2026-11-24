package adminui

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Bulk action vocabulary (T19 F139). Bulk delete is deliberately ABSENT:
// the plan forbids it — a mis-clicked selection must never be able to
// destroy the directory in one request.
const (
	bulkActionResend = "resend"
	bulkActionGrant  = "grant"
)

// bulkOutcomesVisible bounds the per-row outcome list to a readable window
// (the import-result panel pattern); the counts above the list stay exact
// either way, so the panel is never dishonest about scale.
const bulkOutcomesVisible = 20

// bulkOutcome is ONE selected row's result. Honesty contract (F140, K-T19):
// every row reports ok (with what happened) or refused (with the reason) —
// a batch summary alone would hide exactly the rows the admin needs to see.
type bulkOutcome struct {
	ID     string
	Email  string
	OK     bool
	Reason string
}

// countBulkOutcomes splits the outcomes into the two numbers the panel leads
// with: how many succeeded and how many were refused.
func countBulkOutcomes(outcomes []bulkOutcome) (ok, failed int) {
	for _, o := range outcomes {
		if o.OK {
			ok++
			continue
		}
		failed++
	}
	return ok, failed
}

// visibleBulkOutcomes bounds the rendered list; remainingBulkOutcomes says
// how many more the counts already account for.
func visibleBulkOutcomes(outcomes []bulkOutcome) []bulkOutcome {
	if len(outcomes) <= bulkOutcomesVisible {
		return outcomes
	}
	return outcomes[:bulkOutcomesVisible]
}

func remainingBulkOutcomes(outcomes []bulkOutcome) int {
	rest := len(outcomes) - bulkOutcomesVisible
	if rest < 0 {
		return 0
	}
	return rest
}

// bulkOutcomeOKText names what happened to a successful row, per action —
// "ok" alone would not say what the admin's click DID to the user.
func bulkOutcomeOKText(action string) string {
	if action == bulkActionGrant {
		return "role granted"
	}
	return "verification email sent"
}

// userBulk is the bulk bar's POST target (F138/F140). It parses the checked
// ids and the chosen action, runs the action per row through the SAME
// service paths the single-user routes use (F142 — bulk resend is the resend
// route's SendVerificationEmail, bulk grant is the roles route's GrantRoles
// with identical actor attribution), and renders the per-row outcome panel
// as the htmx swap fragment. Refusals render a refusal panel instead of a
// success-shaped one: no ids, an unknown action, or a role outside the
// admin set each answer honestly and dispatch nothing.
func (h *Handler) userBulk(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	ids := r.Form["ids"]
	action := r.FormValue("action")

	if len(ids) == 0 {
		renderPartial(w, r, bulkRefusalPanel("No users selected. Check at least one user, then pick an action."))
		return
	}
	switch action {
	case bulkActionResend:
		renderPartial(w, r, bulkOutcomePanel(action, h.bulkResend(r.Context(), ids)))
	case bulkActionGrant:
		role := identitymodel.Role(strings.TrimSpace(r.FormValue("roles")))
		if !slices.Contains(identitymodel.AdminRoles(), role) {
			renderPartial(w, r, bulkRefusalPanel("Pick a platform role to grant."))
			return
		}
		renderPartial(w, r, bulkOutcomePanel(action, h.bulkGrant(r.Context(), ids, role)))
	default:
		renderPartial(w, r, bulkRefusalPanel("Pick an action: resend verification or grant role."))
	}
}

// bulkResend re-issues the verification email per selected user, the resend
// route's exact service path — including its refusal when verification is
// not configured, which surfaces per row instead of as one opaque failure.
func (h *Handler) bulkResend(ctx context.Context, ids []string) []bulkOutcome {
	outcomes := make([]bulkOutcome, 0, len(ids))
	for _, raw := range ids {
		o := bulkOutcome{ //nolint:exhaustruct // refusal rows fill OK/Email below
			ID: strings.TrimSpace(raw),
		}
		id, err := identitymodel.ParseUserID(o.ID)
		if err != nil {
			o.Reason = "not a valid user id"
			outcomes = append(outcomes, o)
			continue
		}
		if u, err := h.config.Service.GetUser(ctx, id); err == nil {
			o.Email = u.Email
		}
		if _, err := h.config.Service.SendVerificationEmail(ctx, id); err != nil {
			o.Reason = actionErrorMessage(err)
		} else {
			o.OK = true
		}
		outcomes = append(outcomes, o)
	}
	return outcomes
}

// bulkGrant grants ONE platform role to each selected user in the wildcard
// domain — the roles route's exact GrantRoles dispatch (actor attribution
// matches the single grant: ActorIDFromUser(target)).
func (h *Handler) bulkGrant(ctx context.Context, ids []string, role identitymodel.Role) []bulkOutcome {
	outcomes := make([]bulkOutcome, 0, len(ids))
	for _, raw := range ids {
		o := bulkOutcome{ //nolint:exhaustruct // the remaining fields fill in below
			ID: strings.TrimSpace(raw),
		}
		id, err := identitymodel.ParseUserID(o.ID)
		if err != nil {
			o.Reason = "not a valid user id"
			outcomes = append(outcomes, o)
			continue
		}
		if u, err := h.config.Service.GetUser(ctx, id); err == nil {
			o.Email = u.Email
		}
		if err := h.config.Service.GrantRoles(
			ctx,
			identitymodel.ActorIDFromUser(id),
			identitymodel.NewTenantID(identitymodel.WildcardDomain),
			[]identitymodel.Role{role},
		); err != nil {
			o.Reason = actionErrorMessage(err)
		} else {
			o.OK = true
		}
		outcomes = append(outcomes, o)
	}
	return outcomes
}

// userExportIDsCSV / userExportIDsJSON stream the CHECKED selection (T19
// F139): the ids travel as the `ids` query param (comma-separated), the
// wire shape is the Service export's exactly, and the filename says
// "selection" so a browser-saved file tells the truth about its scope.
func (h *Handler) userExportIDsCSV(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.exportIDs(w, r, usermgmt.UserDataFormatCSV)
}

func (h *Handler) userExportIDsJSON(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	h.exportIDs(w, r, usermgmt.UserDataFormatJSON)
}

// exportSelectionIDs parses the ids param, dropping empty segments so a
// trailing comma or double comma cannot manufacture a phantom id.
func exportSelectionIDs(raw string) []string {
	segments := strings.Split(raw, ",")
	ids := make([]string, 0, len(segments))
	for _, s := range segments {
		if s = strings.TrimSpace(s); s != "" {
			ids = append(ids, s)
		}
	}
	return ids
}

func (h *Handler) exportIDs(w http.ResponseWriter, r *http.Request, format usermgmt.UserDataFormat) {
	ids := exportSelectionIDs(r.URL.Query().Get("ids"))
	if len(ids) == 0 {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"No users selected",
			"Check at least one user in the list, then use Export selection.",
		)
		return
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	matched := make([]*identitymodel.User, 0, len(ids))
	for _, u := range h.config.Service.ReadModel().AllUsers() {
		if _, ok := want[u.ID.Get().String()]; ok {
			matched = append(matched, u)
		}
	}
	if len(matched) == 0 {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"None of the selected users exist",
			"Every id in the selection is missing from the directory (the users may have been deleted).",
		)
		return
	}

	name := "users-export-selection." + string(format)
	w.Header().Set("Content-Type", exportContentType(format))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))

	var err error
	switch format {
	case usermgmt.UserDataFormatJSON:
		err = writeUsersJSON(w, matched)
	case usermgmt.UserDataFormatCSV:
		err = writeUsersCSV(w, matched)
	}
	if err != nil {
		// Headers are already committed by the streaming write; the partial
		// output is honest about failing rather than silently truncated.
		triggerToast(w, "err", "Export failed: "+actionErrorMessage(err))
	}
}
