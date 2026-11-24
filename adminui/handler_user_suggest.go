package adminui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"slices"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// suggestLimit caps one suggest response. A type-ahead popup is scanned by
// eye; beyond twenty rows the admin is scrolling, not picking, so the tail is
// cut rather than shipped (the 200-email datalist cap retired with the T08
// combobox — this is its successor, applied per keystroke instead of per
// page render).
const suggestLimit = 20

// errSuggestEmptyQuery is the Rejection for an absent or blank query. An
// empty q would otherwise mean "the whole directory", which is the users
// list's job — the endpoint refuses honestly instead of impersonating it.
var errSuggestEmptyQuery error = errorfamily.NewRejection("adminui.suggest.empty_query", "the q parameter is required and must not be blank")

// userSuggestResult is one suggestion row: enough to label the popup option
// and fill the form field — the id (for future actor-carrying posts), the
// email (the value the members form posts), and the display name (the
// human-readable label).
type userSuggestResult struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// userSuggest serves QUERY + GET /users/suggest: a prefix match on email and
// display name over the same directory read model the users list reads,
// capped at [suggestLimit], rendered as JSON. It is the type-ahead backend
// for the members-add combobox (T08): the page no longer ships a bounded
// slice of the directory; the client asks per keystroke.
//
// The empty-query refusal is honest: returning every user would make this
// endpoint a second users list, and a blank keystroke is a mistake, not a
// browse intent. The panel's client never sends one (it suppresses the fetch
// until two characters), but a direct caller gets a named 400, not silence.
func (h *Handler) userSuggest(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		h.writeJSONRejection(w, http.StatusBadRequest, errSuggestEmptyQuery)
		return
	}
	needle := strings.ToLower(q)

	matches := make([]userSuggestResult, 0, suggestLimit)
	for _, u := range h.config.Service.ReadModel().AllUsers() {
		// A user with no email cannot be picked into an email field.
		if u.Email == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(u.Email), needle) &&
			!strings.HasPrefix(strings.ToLower(u.DisplayName), needle) {
			continue
		}
		matches = append(matches, userSuggestResult{
			ID:          u.ID.Get().String(),
			Email:       u.Email,
			DisplayName: u.DisplayName,
		})
		if len(matches) >= suggestLimit {
			break
		}
	}
	slices.SortFunc(matches, func(a, b userSuggestResult) int {
		return strings.Compare(strings.ToLower(a.Email), strings.ToLower(b.Email))
	})

	w.Header().Set("Content-Type", "application/json")
	enc := jsontext.NewEncoder(w, jsontext.WithIndent("  "))
	if err := json.MarshalEncode(enc, matches); err != nil {
		h.writeErrorPage(w, r, http.StatusInternalServerError, "Suggest failed", "The suggestions could not be rendered.")
	}
}

// writeJSONRejection writes one error-family Rejection as a small JSON
// object. The suggest endpoint is an API, not a page: an HTML error page
// inside a fetch() response would render as garbage in the popup path, so
// refusals travel as JSON too.
func (h *Handler) writeJSONRejection(w http.ResponseWriter, status int, rejection error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := jsontext.NewEncoder(w, jsontext.WithIndent("  "))
	body := map[string]string{"error": rejection.Error()}
	_ = json.MarshalEncode(enc, body) // writing to a ResponseWriter cannot fail meaningfully; a short write surfaces on the client
}
