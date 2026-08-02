package usermgmt

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	httputil "github.com/larsartmann/httputil"
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

// credentialListResponse is the paginated response for credential listing.
//cqrs-lint:ignore(S006) credential metadata, not financial data; encryption applied at event-store layer
type credentialListResponse struct {
	Credentials []credentialSummary `json:"credentials"`
	TotalCount  int                 `json:"total_count"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"page_size"`
	TotalPages  int                 `json:"total_pages"`
}

const (
	credentialPaginationPage     = "page"
	credentialPaginationPageSize = "page_size"
)

func (h *AuthHandler) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
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

	totalCount := len(summaries)
	p := query.NewPagination(
		httputil.ParseUintQuery(r, credentialPaginationPage),
		httputil.ParseUintQuery(r, credentialPaginationPageSize),
	)
	page, pageSize := int(p.Page), int(p.PageSize)
	paged := paginate(summaries, page, pageSize)

	writeJSON(w, http.StatusOK, credentialListResponse{
		Credentials: paged,
		TotalCount:  totalCount,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages(totalCount, pageSize),
	})
}

func totalPages(totalCount, pageSize int) int {
	if pageSize == 0 {
		return 0
	}
	pages := totalCount / pageSize
	if totalCount%pageSize > 0 {
		pages++
	}
	return pages
}

func paginate[T any](items []T, page, pageSize int) []T {
	if len(items) == 0 {
		return items
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []T{}
	}
	end := min(start+pageSize, len(items))
	return items[start:end]
}

func (h *AuthHandler) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	user, encodedID, ok := h.currentUserWithPathValue(w, r, "id", "credential id is required")
	if !ok {
		return
	}

	credID, err := base64.RawURLEncoding.DecodeString(encodedID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credential id encoding")
		return
	}

	if err := h.service.RemoveCredential(r.Context(), user.ID, credID); err != nil {
		writeDispatchError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusRemoved})
}
