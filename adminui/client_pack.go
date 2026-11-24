package adminui

import "encoding/json"

// userCopySnapshot is the JSON payload behind the detail page's
// copy-row-as-JSON support action (T17 F128): the account facts an admin
// pastes into a ticket or an API call, composed server-side from the same
// resolved data the page renders — no client-side re-derivation, no second
// source of truth.
type userCopySnapshot struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Verified    bool   `json:"email_verified"`
	CreatedAt   string `json:"created_at"`
}

// userCopyJSON marshals the snapshot. Marshal of this plain-string/bool
// struct cannot fail in practice; the error arm degrades to an empty object
// rather than shipping a half-written payload attribute. The timestamp is
// UTC (gosmopolitan: no time.Local anywhere).
func userCopyJSON(d userDetailData) string {
	snap := userCopySnapshot{
		ID:          d.User.ID.Get().String(),
		Email:       d.User.Email,
		DisplayName: d.User.DisplayName,
		Verified:    d.User.EmailVerified,
		CreatedAt:   d.User.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	b, err := json.Marshal(snap)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// clientPackSkeletonID is the DOM id of the users list's first-load skeleton
// (T17 F124/F125). Server-rendered markup carries it hidden via the
// .htmx-indicator convention; admin.js removes the node after the first
// swap, so the skeleton can never reappear for already-loaded data.
const clientPackSkeletonID = "users-skeleton"
