package usermgmt

import (
	"encoding/base64"
	"net/http"
	"time"
)

// credentialSummary is a sanitized view of a WebAuthnCredential for API responses.
// Sensitive fields (PublicKey, SignCount, transports, AAGUID) are excluded.
type credentialSummary struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	BackupEligible bool      `json:"backup_eligible"`
	BackupState    bool      `json:"backup_state"`
	CreatedAt      time.Time `json:"created_at"`
}

func (h *AuthHandler) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	summaries := make([]credentialSummary, 0, len(user.Credentials))
	for _, cred := range user.Credentials {
		summaries = append(summaries, credentialSummary{
			ID:             base64.RawURLEncoding.EncodeToString(cred.ID),
			Name:           cred.Name,
			BackupEligible: cred.BackupEligible,
			BackupState:    cred.BackupState,
			CreatedAt:      cred.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"credentials": summaries})
}

func (h *AuthHandler) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	encodedID := r.PathValue("id")
	if encodedID == "" {
		writeError(w, http.StatusBadRequest, "credential id is required")
		return
	}

	credID, err := base64.RawURLEncoding.DecodeString(encodedID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credential id encoding")
		return
	}

	if err := h.service.RemoveCredential(r.Context(), user.ID, credID); err != nil {
		writeError(w, errorStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: "removed"})
}
