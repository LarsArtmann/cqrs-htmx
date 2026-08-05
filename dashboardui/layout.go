package dashboardui

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/templ-components/icons"
)

// brandInitialsLen is the number of characters used for the sidebar brand initials.
const brandInitialsLen = 2

// renderLayout wraps content in the dashboard HTML shell: sidebar,
// header, content area. The layout uses semantic HTML5 landmarks for
// accessibility (aside, nav, main, header) and CSS classes instead of
// inline styles.
func (d *Dashboard) renderLayout(p pageData, content func() string) string {
	// HTMX partial mode: return only the <main> element and a <title> tag.
	// HTMX boost extracts the title for the browser tab and swaps #main-content.
	if p.HTMX {
		return fmt.Sprintf(
			"<title>%s · %s</title><main id=\"main-content\" class=\"content-area\">%s</main>",
			esc(p.Title), esc(p.Brand), content(),
		)
	}

	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\"/>\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"/>\n")
	b.WriteString("<meta name=\"color-scheme\" content=\"light dark\"/>\n")
	b.WriteString("<meta name=\"robots\" content=\"noindex\"/>\n")
	fmt.Fprintf(&b, "<title>%s · %s</title>\n", esc(p.Title), esc(p.Brand))
	fmt.Fprintf(&b, "<style>:root{--accent:%s;}</style>\n", esc(p.Accent))
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%s/-/dashboard.css\"/>\n", p.BasePath)
	fmt.Fprintf(&b, "<script src=\"%s/-/htmx.js\"></script>\n", p.BasePath)

	if p.Caps.EventBus {
		fmt.Fprintf(&b, "<script src=\"%s/-/dashboard.js\"></script>\n", p.BasePath)
	}

	b.WriteString("</head>\n<body>\n")

	b.WriteString(`<a href="#main-content" class="skip-link">Skip to content</a>`)

	b.WriteString(`<div class="app-layout" data-hx-boost="true" data-hx-target="#main-content" data-hx-select="#main-content" data-hx-swap="outerHTML">`)

	b.WriteString(d.renderSidebar(p))
	b.WriteString(`<div class="sidebar-backdrop" data-sidebar-backdrop></div>`)
	b.WriteString(`<div class="app-main">`)
	b.WriteString(d.renderHeader(p))
	fmt.Fprintf(&b, `<main id="main-content" class="content-area">%s</main>`, content())
	b.WriteString("</div></div>\n")

	b.WriteString(d.renderToastContainer())

	b.WriteString("</body>\n</html>")

	return b.String()
}

func (d *Dashboard) renderSidebar(p pageData) string {
	var b strings.Builder

	b.WriteString(`<aside class="sidebar">`)

	fmt.Fprintf(
		&b,
		`<div class="sidebar-brand"><span class="sidebar-badge">%s</span><span class="sidebar-name">%s</span></div>`,
		initials(p.Brand),
		esc(p.Brand),
	)

	b.WriteString(`<nav class="sidebar-nav" aria-label="Dashboard navigation">`)

	for _, item := range p.Nav {
		classes := "nav-link"
		if item.Active {
			classes += " nav-link-active"
		}

		fmt.Fprintf(
			&b,
			`<a href="%s%s" class="%s"><span class="nav-icon" aria-hidden="true">%s</span><span>%s</span></a>`,
			p.BasePath,
			item.Href,
			classes,
			navIconSVG(item.Icon),
			esc(item.Label),
		)
	}

	b.WriteString("</nav>")

	if p.LogoutURL != "" {
		fmt.Fprintf(
			&b,
			`<a href="%s" class="nav-link nav-link-logout">Logout</a>`,
			esc(p.LogoutURL),
		)
	}

	b.WriteString(`<div class="sidebar-footer">dashboardui</div>`)
	b.WriteString("</aside>")

	return b.String()
}

func (d *Dashboard) renderHeader(p pageData) string {
	var indicator string

	if p.Caps.EventBus {
		indicator = `<span class="live-indicator" data-live-indicator aria-label="Live updates status"></span>` +
			`<span class="sse-status" data-sse-status aria-live="polite">Connecting</span>` +
			`<span class="sse-count" data-sse-count aria-live="polite"></span>`
	}

	return fmt.Sprintf(
		`<header class="app-header"><button class="hamburger" aria-label="Toggle navigation menu" data-hamburger aria-expanded="false"><span></span><span></span><span></span></button><div class="header-title">%s%s</div></header>`,
		esc(p.Title),
		indicator,
	)
}

// renderToastContainer renders the hidden toast notification container.
// HTMX write operations dispatch Hx-Trigger events that this container
// listens for and renders as transient toast messages.
func (d *Dashboard) renderToastContainer() string {
	return `<div id="toast-container" class="toast-container" role="region" aria-label="Notifications" aria-live="polite"></div>` +
		`<script>
(function() {
  document.body.addEventListener("showToast", function(e) {
    var d = e.detail || {};
    var c = document.getElementById("toast-container");
    if (!c) return;
    var t = document.createElement("div");
    t.className = "toast toast-" + (d.kind || "ok");
    t.setAttribute("role", "alert");
    t.textContent = d.message || "";
    c.appendChild(t);
    requestAnimationFrame(function() { t.classList.add("toast-visible"); });
    setTimeout(function() {
      t.classList.remove("toast-visible");
      setTimeout(function() { t.remove(); }, 300);
    }, 4000);
  });
})();
</script>`
}

// navIconSVG returns an inline SVG icon for the given icon name using the
// templ-components Heroicons path data. Unknown names fall back to the
// Question icon (a "?" symbol) via the library's built-in fallback.
func navIconSVG(name string) string {
	paths := icons.IconPathData(mapNavIconName(name))
	var b strings.Builder
	b.WriteString(`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="16" height="16" aria-hidden="true">`)
	for _, p := range paths {
		b.WriteString(`<path stroke-linecap="round" stroke-linejoin="round" d="`)
		b.WriteString(p)
		b.WriteString(`"/>`)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

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
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write([]byte(dashboardCSS))
	}
}

// serveJS returns a handler that serves the dashboard JavaScript.
func (d *Dashboard) serveJS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write([]byte(dashboardJS))
	}
}

const dashboardCSS = `
:root {
	--accent: #4f46e5;
	--bg: #f6f7f9;
	--surface: #ffffff;
	--surface-hover: #f0f1f4;
	--text: #0f172a;
	--muted: #64748b;
	--border: #e6e8ec;
	--ok: #16a34a;
	--warn: #d97706;
	--err: #dc2626;
	--sidebar-bg: #0f172a;
	--sidebar-text: #94a3b8;
	--sidebar-active: #f1f5f9;
	--sidebar-width: 248px;
	--radius: 6px;
	--radius-lg: 8px;
	--radius-sm: 4px;
	--gap: 16px;
	--transition: 0.2s ease;
}
@media (prefers-color-scheme: dark) {
	:root {
		--bg: #0b1120;
		--surface: #131c31;
		--surface-hover: #1a2740;
		--text: #e6edf6;
		--muted: #93a4bd;
		--border: #233049;
		--sidebar-bg: #060d1c;
		--sidebar-text: #64748b;
		--sidebar-active: #e6edf6;
	}
}

/* ===== Base ===== */
body { background: var(--bg); color: var(--text); font-family: ui-sans-serif, system-ui, -apple-system, sans-serif; margin: 0; line-height: 1.6; }
* { box-sizing: border-box; }
a { color: var(--accent); text-decoration: none; transition: opacity var(--transition); }
a:hover { opacity: 0.8; }

/* ===== Focus styles (accessibility) ===== */
*:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: var(--radius-sm); }

/* ===== Code ===== */
code { font-family: ui-monospace, monospace; font-size: 0.88em; background: var(--border); padding: 0.1rem 0.35rem; border-radius: var(--radius-sm); }
.mono { font-family: ui-monospace, monospace; font-size: 0.85em; }

/* ===== App layout ===== */
.app-layout { display: grid; grid-template-columns: var(--sidebar-width) 1fr; min-height: 100vh; }
.app-main { display: flex; flex-direction: column; min-width: 0; }

/* ===== Sidebar ===== */
.sidebar { background: var(--sidebar-bg); color: var(--sidebar-text); padding: 18px 14px; position: sticky; top: 0; height: 100vh; overflow-y: auto; display: flex; flex-direction: column; }
.sidebar-brand { display: flex; align-items: center; gap: 10px; padding-bottom: 16px; color: var(--sidebar-active); font-weight: 700; }
.sidebar-badge { display: grid; place-items: center; width: 26px; height: 26px; background: var(--accent); color: white; border-radius: var(--radius-lg); font-size: 0.85em; flex-shrink: 0; }
.sidebar-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sidebar-nav { display: flex; flex-direction: column; gap: 2px; }
.sidebar-footer { margin-top: auto; padding-top: 8px; font-size: 0.75rem; opacity: 0.5; }

/* ===== Navigation links ===== */
.nav-link { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: var(--radius); text-decoration: none; font-size: 0.9rem; font-weight: 500; color: var(--sidebar-text); transition: background var(--transition), color var(--transition); }
.nav-link:hover { background: rgba(255,255,255,0.06); color: var(--sidebar-active); opacity: 1; }
.nav-link-active { background: color-mix(in srgb, var(--accent) 14%, transparent); color: var(--accent); }
.nav-icon { display: flex; align-items: center; justify-content: center; width: 16px; height: 16px; flex-shrink: 0; }
.nav-link-logout { margin-top: 8px; }

/* ===== Header ===== */
.app-header { position: sticky; top: 0; z-index: 5; display: flex; align-items: center; justify-content: space-between; padding: 14px 24px; border-bottom: 1px solid var(--border); background: color-mix(in srgb, var(--surface) 86%, transparent); backdrop-filter: blur(8px); }
.header-title { font-size: 1.1rem; font-weight: 600; display: flex; align-items: center; gap: 8px; }

/* ===== Live indicator ===== */
.live-indicator { display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: var(--ok); opacity: 0.4; transition: opacity 0.3s; }
.sse-status { font-size: 0.75rem; color: var(--muted); }
.sse-count { font-size: 0.75rem; color: var(--muted); font-variant-numeric: tabular-nums; }
.copyable { cursor: pointer; position: relative; }
.copyable:hover::after { content: "📋"; font-size: 0.75em; margin-left: 4px; opacity: 0.6; }

/* ===== Content area ===== */
.content-area { width: 100%; max-width: 1200px; padding: 24px; }

/* ===== Page header ===== */
.page-header { margin-bottom: 24px; }
.page-header h2 { margin: 0 0 4px; }
.page-header .page-subtitle { color: var(--muted); font-size: 0.88em; }

/* ===== Tables ===== */
.data-table { width: 100%; border-collapse: collapse; margin-bottom: 24px; }
.data-table th { padding: 8px; text-align: left; border-bottom: 2px solid var(--border); font-size: 0.85em; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; }
.data-table td { padding: 8px; }
.data-table tbody tr { border-bottom: 1px solid var(--border); transition: background var(--transition); }
.data-table tbody tr:hover { background: var(--surface-hover); }
.data-table tbody tr:nth-child(even) { background: color-mix(in srgb, var(--surface-hover) 50%, transparent); }
.data-table thead th { position: sticky; top: 0; background: var(--surface); z-index: 1; }

/* ===== Stat cards ===== */
.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: var(--gap); margin-bottom: 24px; }
.stat-card { border: 2px solid var(--border); padding: 20px; text-align: center; background: var(--surface); border-radius: var(--radius-lg); transition: border-color var(--transition); }
.stat-card:hover { border-color: var(--accent); }
.stat-card.accent { border-color: var(--accent); }
.stat-card.ok { border-color: var(--ok); }
.stat-card.warn { border-color: var(--warn); }
.stat-card.err { border-color: var(--err); }
.stat-card-value { font-size: 2.5rem; font-weight: 900; line-height: 1; }
.stat-card-label { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.12em; margin-top: 6px; color: var(--muted); }

/* ===== Empty state ===== */
.empty-state { padding: 40px; text-align: center; color: var(--muted); }
.empty-state h2 { margin-bottom: 8px; }

/* ===== Metadata table ===== */
.meta-table { width: 100%; border-collapse: collapse; font-size: 0.88em; }
.meta-table td { padding: 6px 8px; }
.meta-table tr { border-bottom: 1px solid var(--border); }
.meta-key { color: var(--muted); font-weight: 500; }
.meta-val { font-family: ui-monospace, monospace; font-size: 0.85em; }

/* ===== Pre/code blocks ===== */
.code-block { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-lg); padding: 16px; overflow-x: auto; font-size: 0.85em; line-height: 1.5; margin: 0; }
.code-block code { background: none; padding: 0; }

/* ===== Cards/panels ===== */
.panel { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-lg); padding: 16px; margin-bottom: 24px; }
.panel-title { font-weight: 600; margin-bottom: 12px; font-size: 0.95rem; }

/* ===== Grid for two-column layouts ===== */
.two-col-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }

/* ===== Badges/pills ===== */
.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 0.75em; font-weight: 600; }
.badge-ok { background: color-mix(in srgb, var(--ok) 15%, transparent); color: var(--ok); }
.badge-warn { background: color-mix(in srgb, var(--warn) 15%, transparent); color: var(--warn); }
.badge-err { background: color-mix(in srgb, var(--err) 15%, transparent); color: var(--err); }
.badge-neutral { background: var(--border); color: var(--muted); }

/* ===== Buttons ===== */
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 16px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); color: var(--text); cursor: pointer; font-size: 0.85em; font-weight: 500; transition: background var(--transition), border-color var(--transition); text-decoration: none; }
.btn:hover { background: var(--surface-hover); opacity: 1; }
.btn-danger { border-color: var(--err); color: var(--err); background: transparent; }
.btn-danger:hover { background: color-mix(in srgb, var(--err) 8%, transparent); }
.btn-accent { border-color: var(--accent); color: var(--accent); }
.btn-accent:hover { background: color-mix(in srgb, var(--accent) 8%, transparent); }

/* ===== Pagination ===== */
.pagination { display: flex; gap: 4px; flex-wrap: wrap; align-items: center; margin-top: 16px; }
.pagination a, .pagination span { padding: 4px 10px; border: 1px solid var(--border); border-radius: var(--radius-sm); text-decoration: none; font-size: 0.85em; color: var(--muted); }
.pagination a:hover { background: var(--surface-hover); }
.pagination .current { border-color: var(--accent); background: var(--accent); color: white; font-weight: 600; }
.pagination-info { border: none !important; color: var(--muted) !important; font-size: 0.8em !important; margin: 0 4px; }
.page-size-selector { border: none !important; padding: 0 !important; margin-left: auto; }
.page-size-selector select { padding: 2px 6px; border: 1px solid var(--border); border-radius: var(--radius-sm); font-size: 0.8em; background: var(--surface); color: var(--text); }
.sort-header { color: var(--accent) !important; text-decoration: none !important; font-weight: 500; }
.sort-header:hover { text-decoration: underline !important; }

/* ===== Version slider (time-travel) ===== */
.version-links { display: flex; flex-wrap: wrap; gap: 4px; }
.version-slider { width: 100%; max-width: 400px; accent-color: var(--accent); cursor: pointer; }
.version-display { font-size: 0.9rem; color: var(--muted); }

/* ===== Filter bar ===== */
.filter-bar { display: flex; gap: 12px; flex-wrap: wrap; margin-bottom: 16px; align-items: center; }
.filter-bar input, .filter-bar select { padding: 6px 10px; border: 1px solid var(--border); border-radius: var(--radius-sm); font-size: 0.88em; background: var(--surface); color: var(--text); }
.filter-bar label { font-size: 0.85em; font-weight: 500; color: var(--muted); }

/* ===== Toast container ===== */
.toast-container { position: fixed; bottom: 20px; right: 20px; z-index: 9999; display: flex; flex-direction: column; gap: 8px; pointer-events: none; }
.toast { padding: 12px 20px; border-radius: var(--radius); font-size: 0.9rem; font-weight: 500; opacity: 0; transform: translateX(100%); transition: opacity 0.3s, transform 0.3s; pointer-events: auto; min-width: 200px; max-width: 400px; }
.toast-visible { opacity: 1; transform: translateX(0); }
.toast-ok { background: var(--ok); color: white; }
.toast-err { background: var(--err); color: white; }
.toast-warn { background: var(--warn); color: white; }

/* ===== HTMX loading indicator ===== */
.htmx-indicator { display: none; }
.htmx-request .htmx-indicator, .htmx-request.htmx-indicator { display: inline; }
.htmx-request.htmx-indicator-dot::after { content: " ⏳"; }

/* ===== SSE live-update row highlight ===== */
@keyframes newRowHighlight {
  from { background: color-mix(in srgb, var(--accent) 20%, transparent); }
  to { background: transparent; }
}
.new-row { animation: newRowHighlight 2s ease-out; }

/* ===== Hamburger (mobile only) ===== */
.hamburger { display: none; flex-direction: column; justify-content: center; gap: 4px; width: 36px; height: 36px; padding: 6px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); cursor: pointer; }
.hamburger span { display: block; height: 2px; background: var(--text); border-radius: 1px; transition: transform 0.2s, opacity 0.2s; }

/* ===== Sidebar backdrop (mobile only) ===== */
.sidebar-backdrop { display: none; }

/* ===== Table scroll wrapper ===== */
.table-scroll { overflow-x: auto; -webkit-overflow-scrolling: touch; }

/* ===== Responsive ===== */
@media (max-width: 768px) {
	.app-layout { grid-template-columns: 1fr; }
	.sidebar { position: fixed; left: -100%; width: var(--sidebar-width); z-index: 100; transition: left 0.3s; height: 100vh; }
	.sidebar.open { left: 0; }
	.sidebar-backdrop { display: block; position: fixed; inset: 0; background: rgba(0,0,0,0.4); z-index: 99; opacity: 0; pointer-events: none; transition: opacity 0.3s; }
	.sidebar-backdrop.visible { opacity: 1; pointer-events: auto; }
	.hamburger { display: flex; }
	.app-header { padding: 12px 16px; }
	.content-area { padding: 16px; }
	.two-col-grid { grid-template-columns: 1fr; }
	.data-table { font-size: 0.8em; }
	.btn { min-height: 44px; padding: 10px 16px; }
	.filter-bar { flex-direction: column; align-items: stretch; }
	.filter-bar input, .filter-bar select { width: 100%; }
	.nav-link { padding: 12px 10px; font-size: 1rem; }
	.stat-grid { grid-template-columns: 1fr 1fr; gap: 12px; }
	.stat-card { padding: 14px; }
	.stat-card-value { font-size: 1.8rem; }
}

/* ===== Print styles ===== */
@media print {
	.sidebar, .app-header, .toast-container { display: none; }
	.app-layout { display: block; }
	.content-area { max-width: 100%; padding: 0; }
}

/* ===== Animations ===== */
@media (prefers-reduced-motion: reduce) {
	* { transition: none !important; animation: none !important; }
}

/* ===== Skip-to-content link ===== */
.skip-link { position: absolute; left: -999px; top: 0; z-index: 10000; padding: 8px 16px; background: var(--accent); color: white; border-radius: 0 0 var(--radius) 0; }
.skip-link:focus { left: 0; }

/* ===== Utility classes ===== */
.section-gap { margin-bottom: 16px; }
.section-gap-lg { margin-bottom: 24px; }
.cell-emph { font-weight: 600; }
.inline-form { display: inline; }
.num { text-align: right; font-variant-numeric: tabular-nums; }
`

const dashboardJS = `
(function() {
  var scriptSrc = document.currentScript.src;
  var path = scriptSrc.replace(/\/dashboard\.js$/, "");
  var base = path.replace(/\/-\/$/, "");
  var streamUrl = base + "/-/events/stream";
  var indicator = document.querySelector("[data-live-indicator]");
  var statusEl = document.querySelector("[data-sse-status]");
  var countEl = document.querySelector("[data-sse-count]");
  var reconnectDelay = 1000;
  var maxReconnectDelay = 30000;
  var eventCount = 0;
  var es = null;
  var reconnectTimer = null;

  function updateStatus(state) {
    var labels = { connecting: "Connecting", open: "Live", error: "Reconnecting", closed: "Disconnected" };
    if (statusEl) statusEl.textContent = labels[state] || state;
    if (indicator) {
      indicator.style.opacity = state === "open" ? "1" : "0.4";
      indicator.title = labels[state] || state;
    }
  }

  function updateCount() {
    if (countEl) countEl.textContent = eventCount + " events";
  }

  function handleEvent(e) {
    try {
      var data = JSON.parse(e.data);
      document.dispatchEvent(new CustomEvent("dashboard:event", { detail: data }));
      eventCount++;
      updateStatus("open");
      updateCount();
    } catch (err) {}
  }

  function connect() {
    if (es) es.close();
    if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }

    es = new EventSource(streamUrl);
    es.addEventListener("event", handleEvent);

    es.onopen = function() {
      reconnectDelay = 1000;
      updateStatus("open");
    };

    es.onerror = function() {
      updateStatus("error");
      es.close();
      reconnectTimer = setTimeout(function() {
        reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
        connect();
      }, reconnectDelay);
    };
  }

  document.addEventListener("visibilitychange", function() {
    if (document.visibilityState === "visible" && (!es || es.readyState === EventSource.CLOSED)) {
      reconnectDelay = 1000;
      connect();
    }
  });

  window.addEventListener("beforeunload", function() {
    if (es) es.close();
  });

  connect();
})();

document.addEventListener("dashboard:event", function(e) {
  var data = e.detail || {};

  if (window.htmx && document.getElementById("projection-health")) {
    htmx.trigger("#projection-health", "refresh");
  }

  var tbody = document.querySelector("#main-content .data-table tbody");
  if (!tbody || !data.eventId) return;

  var script = document.querySelector("script[src$='dashboard.js']");
  var basePath = script ? script.src.replace(/\/-\/dashboard\.js$/, "") : "";
  var now = data.occurredAt ? new Date(data.occurredAt) : new Date();
  var row = document.createElement("tr");
  row.className = "new-row";
  var typeLink = basePath
    ? '<a href="' + basePath + '/events/' + data.eventId + '"><code>' + (data.type || "") + '</code></a>'
    : '<code>' + (data.type || "") + '</code>';
  row.innerHTML = '<td class="mono">' + now.toLocaleTimeString() + '</td><td>' + typeLink + '</td>';
  if (data.streamId) row.innerHTML += '<td class="mono">' + data.streamId.substring(0, 20) + '</td>';
  if (data.version) row.innerHTML += '<td>' + data.version + '</td>';
  tbody.insertBefore(row, tbody.firstChild);
  while (tbody.children.length > 50) tbody.removeChild(tbody.lastChild);
});

document.addEventListener("click", function(e) {
  var el = e.target.closest("[data-copyable]");
  if (!el) return;
  var text = el.getAttribute("data-copyable");
  if (text === "") text = el.textContent.trim();
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(function() {
      document.body.dispatchEvent(new CustomEvent("showToast", { detail: { kind: "ok", message: "Copied to clipboard" } }));
    }).catch(function() {});
  }
});

document.addEventListener("click", function(e) {
  var hamburger = e.target.closest("[data-hamburger]");
  if (hamburger) {
    var sidebar = document.querySelector(".sidebar");
    var backdrop = document.querySelector("[data-sidebar-backdrop]");
    if (sidebar) {
      var isOpen = sidebar.classList.toggle("open");
      hamburger.setAttribute("aria-expanded", isOpen ? "true" : "false");
      if (backdrop) backdrop.classList.toggle("visible", isOpen);
    }
    return;
  }
  var sidebar = document.querySelector(".sidebar");
  if (sidebar && sidebar.classList.contains("open") && !sidebar.contains(e.target)) {
    sidebar.classList.remove("open");
    hamburger = document.querySelector("[data-hamburger]");
    if (hamburger) hamburger.setAttribute("aria-expanded", "false");
    var backdrop = document.querySelector("[data-sidebar-backdrop]");
    if (backdrop) backdrop.classList.remove("visible");
  }
});`
