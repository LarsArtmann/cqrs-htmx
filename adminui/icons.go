package adminui

import (
	"strings"

	"github.com/larsartmann/templ-components/icons"
)

// Adminui icon key. These map adminui's string-based icon keys to the
// typed icons.Name constants from templ-components, so typos fall back to
// the Question icon instead of breaking the UI.
const (
	iconDashboard = string(icons.Chart)
	iconUsers     = string(icons.Users)
	iconTenants   = string(icons.BuildingOffice2)
	iconMembers   = string(icons.UserPlus)
	iconAudit     = string(icons.Clock)
)

// iconSVG returns a complete inline SVG (24x24, stroke = currentColor) for the
// given icon key, used throughout the panel via @templ.Raw. Unknown keys
// render the Question fallback icon (provided by templ-components).
func iconSVG(name string) string {
	paths := icons.IconPathData(icons.Name(name))
	var inner strings.Builder
	for _, d := range paths {
		inner.WriteString(`<path d="`)
		inner.WriteString(d)
		inner.WriteString(`"/>`)
	}
	return `<svg class="h-[18px] w-[18px]" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + inner.String() + `</svg>`
}
