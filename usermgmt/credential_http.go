package usermgmt

import (
	"encoding/base64"
	"net/http"
	"strconv"
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

// credentialListResponse is the paginated response for credential listing.
type credentialListResponse struct {
	Credentials []credentialSummary `json:"credentials"`
	TotalCount  int                 `json:"total_count"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"page_size"`
	TotalPages  int                 `json:"total_pages"`
}

const (
	defaultCredentialPageSize    = 20
	maxCredentialPageSize        = 100
	credentialPaginationPage     = "page"
	credentialPaginationPageSize = "page_size"
)

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

	totalCount := len(summaries)
	page, pageSize := parsePaginationParams(r, totalCount)
	paged := paginate(summaries, page, pageSize)

	writeJSON(w, http.StatusOK, credentialListResponse{
		Credentials: paged,
		TotalCount:  totalCount,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages(totalCount, pageSize),
	})
}

func parsePaginationParams(r *http.Request, totalCount int) (page, pageSize int) {
	page = 1
	pageSize = defaultCredentialPageSize
	if v := r.URL.Query().Get(credentialPaginationPage); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if v := r.URL.Query().Get(credentialPaginationPageSize); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	if pageSize > maxCredentialPageSize {
		pageSize = maxCredentialPageSize
	}
	maxPage := totalPages(totalCount, pageSize)
	if maxPage > 0 && page > maxPage {
		page = maxPage
	}
	return page, pageSize
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
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
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
	writeJSON(w, http.StatusOK, map[string]string{statusKey: statusRemoved})
}
