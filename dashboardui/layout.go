package dashboardui

import (
	"net/http"
	"strings"

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

func initials(brand string) string {
	words := strings.Fields(brand)
	if len(words) == 0 {
		return "?"
	}

	if len(words) == 1 {
		return strings.ToUpper(brand[:min(brandInitialsLen, len(brand))])
	}

	return strings.ToUpper(string(words[0][0]) + string(words[1][0]))
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

