package dashboardui

import (
	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/icons"
)

// LayoutFunc renders one full dashboard page inside a consumer-provided
// shell. It is the embed seam: set [Config.Layout] and the dashboard stops
// owning the document — no sidebar, no header, no theme script, no stylesheet
// links of its own. Instead, every full-page render hands the page metadata
// and the ready-rendered content to this function, and whatever it returns
// IS the HTTP response.
//
// A minimal integration into a consumer app shell:
//
//	cfg.Layout = func(meta dashboardui.PageMeta, content templ.Component) templ.Component {
//	    return myShell(myPage{Title: meta.FullTitle, User: currentUser}, content)
//	}
//
// Contract:
//   - The function owns the ENTIRE document (html/head/body) and must link
//     meta.CSSURLs for the content markup to style, plus meta.ScriptURLs
//     unless the shell already serves its own htmx (never load htmx twice).
//   - meta.Nonce carries the per-request CSP nonce for any script tags the
//     shell emits.
//   - meta.Nav is the capability-filtered navigation; render it in the
//     consumer's sidebar, drop it, or replace it with consumer nav entries —
//     all valid. Nav Hrefs are relative to meta.BasePath.
//   - Write actions surface toasts via the "dashboardui:toast" HX-Trigger
//     event; include a feedback.ToastContainer in the shell to display them.
//   - HTMX partial swaps (boost fragments, polled regions) bypass the shell
//     entirely; no per-request work is needed for them.
//
// Error/404 responses keep the dashboard's minimal built-in error shell —
// they do not flow through LayoutFunc.
type LayoutFunc func(meta PageMeta, content templ.Component) templ.Component

// PageMeta describes one dashboard page render handed to a [LayoutFunc].
type PageMeta struct {
	// Title is the page name without the brand ("Events", "Overview").
	Title string

	// FullTitle is the built-in browser-tab text ("Events · Brand").
	FullTitle string

	// Brand is the dashboard brand ([Config.Title]).
	Brand string

	// BasePath is the trimmed mount path ("/cqrs"), for link and asset URLs.
	BasePath string

	// Nav is the capability-filtered sidebar navigation. Href is relative
	// to BasePath ("/", "/events"); Active marks the current page; Icon is
	// a templ-components icon name.
	Nav []NavLink

	// CSSURLs are the absolute stylesheet URLs the content markup requires
	// (dashboard.css, dashboard-tw.css). A consumer shell must link them
	// (or vendor equivalents) or the panels render unstyled.
	CSSURLs []string

	// ScriptURLs are the absolute script URLs the built-in shell loads
	// (htmx.js, dashboard.js). Skip htmx.js when the shell already loads
	// htmx itself.
	ScriptURLs []string

	// AccentColor is the configured highlight color.
	AccentColor string

	// Nonce is the per-request CSP nonce ("" when no nonce middleware).
	Nonce string

	// ReadOnly mirrors [Config.ReadOnly].
	ReadOnly bool

	// Capabilities reports which panels are active.
	Capabilities Capabilities

	// HTMX is true for partial renders; those bypass the layout entirely,
	// so a LayoutFunc normally never sees HTMX=true.
	HTMX bool
}

// NavLink is one sidebar navigation entry exposed to a [LayoutFunc].
type NavLink struct {
	// Href is relative to BasePath ("/", "/events", "/projections").
	Href string

	// Label is the display text ("Events").
	Label string

	// Icon is the templ-components icon name for the entry.
	Icon icons.Name

	// Active marks the entry matching the current page.
	Active bool
}

// assetURLs are the per-page asset URLs passed to a custom layout. They are
// built from BasePath so the consumer shell can link the dashboard styles
// and scripts regardless of where the panel is mounted.
func assetURLs(basePath string) ([]string, []string) {
	return []string{
			basePath + "/-/dashboard.css",
			basePath + "/-/dashboard-tw.css",
		}, []string{
			basePath + "/-/htmx.js",
			basePath + "/-/dashboard.js",
		}
}

// meta converts the internal page data into the exported shape a
// [LayoutFunc] receives. Nav entries are mapped to their templ-components
// icon names here so consumers never see the internal icon strings.
func (p pageData) meta() PageMeta {
	nav := make([]NavLink, len(p.Nav))
	for i, n := range p.Nav {
		nav[i] = NavLink{
			Href:   n.Href,
			Label:  n.Label,
			Icon:   mapNavIconName(n.Icon),
			Active: n.Active,
		}
	}

	css, scripts := assetURLs(p.BasePath)

	return PageMeta{
		Title:        p.Title,
		FullTitle:    p.Title + " · " + p.Brand,
		Brand:        p.Brand,
		BasePath:     p.BasePath,
		Nav:          nav,
		CSSURLs:      css,
		ScriptURLs:   scripts,
		AccentColor:  p.Accent,
		Nonce:        p.Nonce,
		ReadOnly:     p.ReadOnly,
		Capabilities: p.Caps,
		HTMX:         p.HTMX,
	}
}
