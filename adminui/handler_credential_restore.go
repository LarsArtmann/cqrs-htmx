package adminui

import (
	"encoding/json"
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// F170 (T23 hand-off): credential restore from backup JSON. The F170 UI work
// was cut by the T23 agent because the POST route lives in routes.go; this
// file carries the route's handler so the surface is complete and testable
// without touching users.templ (the T23 agent's file). The user-detail card
// that posts to it folds into a later task; the confirm copy below is the
// danger-gated escalation the card must carry (data-confirm).

// credentialRestoreConfirm is the danger-gated escalation copy (F170): a
// restored credential is a FULLY TRUSTED authentication factor minted from a
// file, not a browser ceremony — the confirm must say what trusting backup
// JSON means before the admin commits.
func credentialRestoreConfirm(userEmail string) string {
	return "Restore a passkey from backup JSON for " + userEmail + "? " +
		"The uploaded credential becomes a trusted sign-in factor immediately — " +
		"restore it only from a backup you control."
}

// userRestoreCredential restores one WebAuthn credential from the uploaded
// backup JSON (F170). identitymodel.WebAuthnCredential is fully JSON-tagged
// and usermgmt.Service.AddCredential is published (service_misc.go), so no
// browser ceremony is needed. Per-error honesty: every refusal names WHAT
// failed (unreadable JSON, empty credential id, service rejection) — the
// admin must know whether the file, the content, or the store refused.
func (h *Handler) userRestoreCredential(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	back := h.config.BasePath + "/users/" + r.PathValue("id")
	target, err := identitymodel.ParseUserID(r.PathValue("id"))
	if err != nil {
		triggerToast(w, "err", "Invalid user id. Nothing restored.")
		redirect(w, r, back)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.writeErrorPage(w, r, http.StatusBadRequest, "Invalid form", "The submitted form could not be parsed.")
		return
	}
	raw := strings.TrimSpace(r.FormValue("backup"))
	if raw == "" {
		triggerToast(w, "err", "No backup JSON was uploaded. Nothing restored.")
		redirect(w, r, back)
		return
	}
	var cred identitymodel.WebAuthnCredential
	if err := json.Unmarshal([]byte(raw), &cred); err != nil {
		triggerToast(w, "err", "The backup file is not valid credential JSON: "+err.Error()+". Nothing restored.")
		redirect(w, r, back)
		return
	}
	if len(cred.ID) == 0 {
		triggerToast(w, "err", "The backup JSON carries no credential id. Nothing restored.")
		redirect(w, r, back)
		return
	}
	if err := h.config.Service.AddCredential(r.Context(), usermgmt.UserID(target), cred); err != nil {
		triggerToast(w, "err", "Credential restore failed: "+actionErrorMessage(err)+". Nothing restored.")
		redirect(w, r, back)
		return
	}
	name := cred.Name
	if name == "" {
		name = "unnamed passkey"
	}
	triggerToast(w, "ok", "Credential restored: "+name+" is a trusted sign-in factor for this user again.")
	redirect(w, r, back)
}
