package adminui

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/templ-components/errorpage"
)

// errorFamilyFor maps an HTTP status to the errorpage visual family.
func errorFamilyFor(status int) errorpage.Family {
	switch status {
	case http.StatusConflict:
		return errorpage.FamilyConflict
	case http.StatusServiceUnavailable:
		return errorpage.FamilyTransient
	case http.StatusInternalServerError:
		return errorpage.FamilyCorruption
	}
	if status >= 500 {
		return errorpage.FamilyInfrastructure
	}
	return errorpage.FamilyRejection
}

// tenantGoneMessage is shared by the three tenant lifecycle codes.
const tenantGoneMessage = "This tenant no longer exists. Refresh the list and try again."

// actionErrorMessages maps known domain error codes to user-safe guidance.
// Codes come from the usermgmt deciders; unknown codes fall back to a
// family-level message. Raw err.Error() text is never rendered to users — it
// leaks internal detail (stream IDs, storage errors) that helps no one.
//
//nolint:gochecknoglobals // static lookup table, never mutated
var actionErrorMessages = map[string]string{
	"usermgmt.tenant.already_exists":         "A tenant with this ID already exists. Pick a different identifier.",
	"usermgmt.tenant.name_required":          "A tenant name is required.",
	"usermgmt.tenant_suspend.not_found":      tenantGoneMessage,
	"usermgmt.tenant_reactivate.not_found":   tenantGoneMessage,
	"usermgmt.tenant_delete.not_found":       tenantGoneMessage,
	"usermgmt.tenant_delete.already_deleted": "This tenant is already deleted.",
	"usermgmt.email_exists":                  "That email address is already registered.",
	"usermgmt.user_not_found":                "That user no longer exists. Refresh the list and try again.",
	"usermgmt.user_id_exists":                "That user ID is already taken.",
	"usermgmt.membership.already_exists":     "That member is already in this tenant. Change their role instead.",
	"usermgmt.membership.actor_required":     "A valid member identifier is required.",
	"usermgmt.membership_roles.not_found":    "That membership no longer exists. Refresh the page.",
	"usermgmt.membership_remove.not_found":   "That membership no longer exists. Refresh the page.",
	"usermgmt.validation":                    "Some values are invalid. Check the form and try again.",
	"usermgmt.oauth_not_configured":          "OAuth2 is not configured on this deployment. Unlinking external accounts requires a provider configuration.",
}

// actionErrorFallback returns the family-level fallback message shown when
// the error code is not in the table.
func actionErrorFallback(f errorfamily.Family) string {
	switch f {
	case errorfamily.Rejection:
		return "The request was rejected. Check the values and try again."
	case errorfamily.Conflict:
		return "The record changed since you loaded it. Refresh and try again."
	case errorfamily.Transient:
		return "The service is temporarily unavailable. Try again in a moment."
	case errorfamily.Corruption:
		return "Something is wrong with the stored data. This has been logged for investigation."
	case errorfamily.Infrastructure:
		return "A backing service failed. Try again in a moment."
	case errorfamily.Orchestration:
		return "A background process is still settling. Refresh and try again shortly."
	default:
		return "The request failed. Try again."
	}
}

// actionErrorStatus maps an errorfamily Family to its HTTP status (mirrors
// errorpage.FamilyStatusCode).
func actionErrorStatus(f errorfamily.Family) int {
	switch f {
	case errorfamily.Rejection:
		return http.StatusBadRequest
	case errorfamily.Conflict:
		return http.StatusConflict
	case errorfamily.Transient:
		return http.StatusServiceUnavailable
	case errorfamily.Corruption:
		return http.StatusInternalServerError
	case errorfamily.Infrastructure:
		return http.StatusServiceUnavailable
	case errorfamily.Orchestration:
		return http.StatusAccepted
	default:
		return http.StatusInternalServerError
	}
}

// actionErrorMessage returns the user-safe message for a failed action —
// used where the flow redirects back with a toast instead of rendering an
// error page (member add/remove/role-update).
func actionErrorMessage(err error) string {
	if message, ok := actionErrorMessages[errorfamily.Code(err)]; ok {
		return message
	}
	return actionErrorFallback(errorfamily.Classify(err))
}

// writeActionError reports a failed write action with a user-safe message:
// a toast for the immediate feedback loop plus a themed error page (bare for
// HTMX swaps). The raw error is logged server-side with its code, never shown.
func (h *Handler) writeActionError(w http.ResponseWriter, r *http.Request, action string, err error) {
	family := errorfamily.Classify(err)
	code := errorfamily.Code(err)
	message, known := actionErrorMessages[code]
	if !known {
		message = actionErrorFallback(family)
	}
	slog.WarnContext(r.Context(), "adminui: action failed",
		"action", action, "family", family.String(), "code", code, "error", err)
	triggerToast(w, "err", action+" failed: "+message)
	h.writeErrorPage(w, r, actionErrorStatus(family), action+" failed", message)
}

// isHTMXRequest reports whether the request comes from an HTMX swap (in which
// case error responses render as bare cards fitting the swap target, not as
// full documents).
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("Hx-Request") == "true"
}

// writeErrorPage writes a templ-components error page instead of a bare
// text/plain http.Error body: the ErrorPage card for HTMX swaps, the card
// inside the panel Layout for navigations. The HTTP status is preserved.
func (h *Handler) writeErrorPage(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	props := errorpage.ErrorPageProps{
		Family:     errorFamilyFor(status),
		StatusCode: status,
		Title:      title,
		Message:    message,
		WayOut:     "Back to dashboard",
		WayOutHref: h.config.BasePath + "/",
	}
	var b strings.Builder
	if err := errorpage.ErrorPage(props).Render(r.Context(), &b); err != nil {
		http.Error(w, title, status)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if isHTMXRequest(r) {
		_, _ = w.Write([]byte(b.String()))
		return
	}
	page := h.page(title, "", nil, r)
	if err := Layout(page, templ.Raw(b.String())).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "adminui: render error page", "error", err)
	}
}

// notFoundHandler serves the styled 404 for any unmatched GET route under the
// panel: the bare NotFound404 view for HTMX swaps, inside the panel Layout
// otherwise.
func (h *Handler) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	props := errorpage.DefaultNotFound404Props()
	props.GoHomeHref = h.config.BasePath + "/"
	props.GoHomeText = "Back to dashboard"
	var b strings.Builder
	if err := errorpage.NotFound404(props).Render(r.Context(), &b); err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	if isHTMXRequest(r) {
		_, _ = w.Write([]byte(b.String()))
		return
	}
	page := h.page("Page not found", "", nil, r)
	if err := Layout(page, templ.Raw(b.String())).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "adminui: render 404 page", "error", err)
	}
}
