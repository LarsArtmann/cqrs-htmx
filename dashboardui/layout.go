package dashboardui

import (
	"net/http"
	"strings"
	"unicode/utf8"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/templ-components/icons"
)

// brandInitialsLen is the number of characters used for the sidebar brand initials.
const brandInitialsLen = 2

// Retry/announcer settings for the library's htmx.GlobalErrorHandling (the
// library's own defaults, pinned here to keep the dashboard contract explicit).
const (
	errorHandlingMaxHistory = 10
	errorHandlingMaxRetries = 2
	errorHandlingRetryMS    = 1000
)

// mapNavIconName translates dashboardui internal icon names to the
// templ-components icons.Name constants.
func mapNavIconName(name string) icons.Name {
	switch name {
	case "chart":
		return icons.Chart
	case "queue":
		return icons.QueueList
	case "cube":
		return icons.Cube
	case "arrow-path":
		return icons.ArrowPath
	case "bug":
		return icons.BugAnt
	case "clipboard":
		return icons.Clipboard
	case "magnifying-glass":
		return icons.Search
	case "clock":
		return icons.Clock
	case "archive":
		return icons.ArchiveBox
	case "signal":
		return icons.Signal
	default:
		return icons.Question
	}
}

// firstRune returns s's first rune as a string, "?" when s does not decode
// as UTF-8 (never a mojibake byte).
func firstRune(s string) string {
	r, _ := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return "?"
	}

	return string(r)
}

func initials(brand string) string {
	words := strings.Fields(brand)
	if len(words) == 0 {
		return "?"
	}

	if len(words) == 1 {
		runes := []rune(words[0])
		if len(runes) > brandInitialsLen {
			runes = runes[:brandInitialsLen]
		}

		return strings.ToUpper(string(runes))
	}

	return strings.ToUpper(firstRune(words[0]) + firstRune(words[1]))
}

// serveCSS returns a handler that serves the dashboard stylesheet.
func (d *Dashboard) serveCSS() http.HandlerFunc {
	return serveConstAsset("dashboard.css", "text/css; charset=utf-8", dashboardCSS)
}

// serveJS returns a handler that serves the dashboard JavaScript.
func (d *Dashboard) serveJS() http.HandlerFunc {
	return serveConstAsset("dashboard.js", "text/javascript; charset=utf-8", dashboardJS)
}

// serveConstAsset serves an in-source constant asset through the root
// module's immutable-asset posture — one caching rule for every
// dashboard-served asset ([newAssetHandler] rides the same
// [cqrshtmx.ServeAsset]).
func serveConstAsset(name, contentType, body string) http.HandlerFunc {
	return cqrshtmx.ServeAsset(name, contentType, []byte(body)).ServeHTTP
}
