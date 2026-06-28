package usermgmt

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/query/v3"
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
	p := query.NewPagination(
		parseUintQueryParam(r, credentialPaginationPage),
		parseUintQueryParam(r, credentialPaginationPageSize),
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

// parseUintQueryParam extracts a uint from a query parameter, returning 0 on
// missing or invalid values (query.NewPagination then applies defaults).
func parseUintQueryParam(r *http.Request, key string) uint {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0
	}
	return uint(n)
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
