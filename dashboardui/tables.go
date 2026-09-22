package dashboardui

import (
	"github.com/larsartmann/templ-components/display"
)

// plainHeaders builds non-sortable typed headers from labels. An empty label
// renders the headerless actions column convention.
func plainHeaders(labels ...string) []display.TableHeader {
	headers := make([]display.TableHeader, 0, len(labels))

	for _, l := range labels {
		headers = append(headers, display.TableHeader{
			Label:         l,
			Sortable:      false,
			SortDirection: "",
			Href:          "",
		})
	}

	return headers
}
