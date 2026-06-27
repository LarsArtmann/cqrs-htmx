package adminui

import "strings"

// Icon keys. Use these constants instead of raw strings so a typo can't silently
// render the fallback dot. Each maps to an entry in iconPaths below.
const (
	iconDashboard = "dashboard"
	iconUsers     = "users"
	iconTenants   = "tenants"
	iconMembers   = "members"
	iconAudit     = "audit"
)

// iconPaths maps adminui icon keys to inline SVG path data (Heroicons outline,
// 24x24 viewBox, stroke = currentColor). Multi-path icons use "|" as a
// separator. Inlining the 18 glyphs adminui needs keeps the module
// self-contained — no external icon-package dependency and no build-time
// coupling to unreleased sibling repos. Source: Heroicons (MIT), the same set
// exposed by larsartmann/templ-components.
var iconPaths = map[string]string{
	"dashboard": "M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z",
	"users":     "M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z",
	"tenants":   "M2.25 21H21.75M3.75 3V21M14.25 3V21M20.25 7.5V21M6.75 6.75H7.5M6.75 9.75H7.5M6.75 12.75H7.5M10.5 6.75H11.25M10.5 9.75H11.25M10.5 12.75H11.25M6.75 21V17.625C6.75 17.0037 7.25368 16.5 7.875 16.5H10.125C10.7463 16.5 11.25 17.0037 11.25 17.625V21M3 3H15M14.25 7.5H21M17.25 11.25H17.2575V11.2575H17.25V11.25ZM17.25 14.25H17.2575V14.2575H17.25V14.25ZM17.25 17.25H17.2575V17.2575H17.25V17.25Z",
	"members":   "M18 7.5v3m0 0v3m0-3h3m-3 0h-3m-2.25-4.125a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zM3 19.235v-.11a6.375 6.375 0 0112.75 0v.109A12.318 12.318 0 019.374 21c-2.331 0-4.512-.645-6.374-1.766z",
	"audit":     "M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z",
	"search":    "M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z",
	"logout":    "M15.75 9V5.25C15.75 4.00736 14.7426 3 13.5 3L7.5 3C6.25736 3 5.25 4.00736 5.25 5.25L5.25 18.75C5.25 19.9926 6.25736 21 7.5 21H13.5C14.7426 21 15.75 19.9926 15.75 18.75V15M18.75 15L21.75 12M21.75 12L18.75 9M21.75 12L9 12",
	"menu":      "M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5",
	"shield":    "M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z",
	"key":       "M15.75 5.25C17.4069 5.25 18.75 6.59315 18.75 8.25M21.75 8.25C21.75 11.5637 19.0637 14.25 15.75 14.25C15.3993 14.25 15.0555 14.2199 14.7213 14.1622C14.1583 14.0649 13.562 14.188 13.158 14.592L10.5 17.25H8.25V19.5H6V21.75H2.25V18.932C2.25 18.3352 2.48705 17.7629 2.90901 17.341L9.408 10.842C9.81202 10.438 9.93512 9.84172 9.83785 9.2787C9.7801 8.94446 9.75 8.60074 9.75 8.25C9.75 4.93629 12.4363 2.25 15.75 2.25C19.0637 2.25 21.75 4.93629 21.75 8.25Z",
	"mail":      "M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75",
	"trash":     "M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0",
	"check":     "M4.5 12.75l6 6 9-13.5",
	"alert":     "M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.664-.833-2.464 0L4.35 16.5c-.77.833.192 2.5 1.732 2.5z",
	"plus":      "M12 4.5v15m7.5-7.5h-15",
	"bot":       "M21.75 17.25v-.228a4.5 4.5 0 00-.12-1.03l-2.268-9.64a3.375 3.375 0 00-3.285-2.602H7.923a3.375 3.375 0 00-3.285 2.602l-2.268 9.64a4.5 4.5 0 00-.12 1.03v.228m19.5 0a3 3 0 01-3 3H5.25a3 3 0 01-3-3m19.5 0a3 3 0 00-3-3H5.25a3 3 0 00-3 3m16.5 0h.008v.008h-.008v-.008zm-3 0h.008v.008h-.008v-.008z",
	"ban":       "M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636",
	"clock":     "M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z",
	"box":       "m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9",
}

// iconSVG returns a complete inline SVG (24x24, stroke = currentColor) for the
// given icon key, used throughout the panel via @templ.Raw. Unknown keys render
// a neutral dot so the UI never breaks.
func iconSVG(name string) string {
	data, ok := iconPaths[name]
	if !ok {
		return `<svg class="admin-nav__icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="3"/></svg>`
	}
	var inner strings.Builder
	for _, d := range strings.Split(data, "|") {
		inner.WriteString(`<path d="`)
		inner.WriteString(d)
		inner.WriteString(`"/>`)
	}
	return `<svg class="admin-nav__icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + inner.String() + `</svg>`
}
