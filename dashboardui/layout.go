package dashboardui

import (
	"fmt"
	"net/http"
	"strings"
)

// brandInitialsLen is the number of characters used for the sidebar brand initials.
const brandInitialsLen = 2

// renderLayout wraps content in the dashboard HTML shell: sidebar,
// header, content area. This is intentionally simple Go-generated HTML
// for the initial version. Future iterations will use templ components.
func (d *Dashboard) renderLayout(p pageData, content func() string) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\"/>\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"/>\n")
	b.WriteString("<meta name=\"color-scheme\" content=\"light dark\"/>\n")
	b.WriteString("<meta name=\"robots\" content=\"noindex\"/>\n")
	fmt.Fprintf(&b, "<title>%s · %s</title>\n", p.Title, p.Brand)
	fmt.Fprintf(&b, "<style>:root{--accent:%s;}</style>\n", p.Accent)
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%s/-/dashboard.css\"/>\n", p.BasePath)
	fmt.Fprintf(&b, "<script src=\"%s/-/htmx.js\"></script>\n", p.BasePath)

	if p.Caps.EventBus {
		fmt.Fprintf(&b, "<script src=\"%s/-/dashboard.js\"></script>\n", p.BasePath)
	}

	b.WriteString("</head>\n<body>\n")

	b.WriteString(`<div style="display:grid;grid-template-columns:248px 1fr;min-height:100vh">`)

	b.WriteString(d.renderSidebar(p))
	b.WriteString(`<div style="display:flex;flex-direction:column;min-width:0">`)
	b.WriteString(d.renderHeader(p))
	fmt.Fprintf(&b, `<main style="width:100%%;max-width:1200px;padding:24px">%s</main>`, content())
	b.WriteString("</div></div>\n")

	b.WriteString("</body>\n</html>")

	return b.String()
}

func (d *Dashboard) renderSidebar(p pageData) string {
	var b strings.Builder
	b.WriteString(
		`<aside style="background:#0f172a;color:#94a3b8;padding:18px 14px;position:sticky;top:0;height:100vh;overflow-y:auto">`,
	)

	fmt.Fprintf(
		&b,
		`<div style="display:flex;align-items:center;gap:10px;padding-bottom:16px;color:#f1f5f9;font-weight:700">
		<span style="display:grid;place-items:center;width:26px;height:26px;background:var(--accent);color:white;border-radius:8px;font-size:0.85em">%s</span>
		%s</div>`,
		initials(p.Brand),
		p.Brand,
	)

	b.WriteString(`<nav style="display:flex;flex-direction:column;gap:2px">`)

	for _, item := range p.Nav {
		bgColor := "transparent"
		color := "#94a3b8"

		if item.Active {
			bgColor = "color-mix(in srgb, var(--accent) 14%, transparent)"
			color = "var(--accent)"
		}

		fmt.Fprintf(
			&b,
			`<a href="%s%s" style="display:flex;align-items:center;gap:10px;padding:8px 10px;border-radius:6px;text-decoration:none;font-size:0.9rem;font-weight:500;color:%s;background:%s">%s</a>`,
			p.BasePath,
			item.Href,
			color,
			bgColor,
			item.Label,
		)
	}

	b.WriteString("</nav>")

	b.WriteString(`<div style="margin-top:auto;padding-top:8px;font-size:0.75rem;opacity:0.5">dashboardui</div>`)
	b.WriteString("</aside>")

	return b.String()
}

func (d *Dashboard) renderHeader(p pageData) string {
	var indicator string

	if p.Caps.EventBus {
		indicator = `<span data-live-indicator style="display:inline-block;width:8px;height:8px;border-radius:50%;background:var(--ok);opacity:0.4;transition:opacity 0.3s;margin-left:8px" title="Live"></span>`
	}

	return fmt.Sprintf(
		`<header style="position:sticky;top:0;z-index:5;display:flex;align-items:center;justify-content:space-between;padding:14px 24px;border-bottom:1px solid #e6e8ec;background:color-mix(in srgb, white 86%%, transparent);backdrop-filter:blur(8px)">
		<div style="font-size:1.1rem;font-weight:600">%s%s</div>
	</header>`,
		p.Title,
		indicator,
	)
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
	--text: #0f172a;
	--muted: #64748b;
	--border: #e6e8ec;
	--ok: #16a34a;
	--warn: #d97706;
	--err: #dc2626;
}
@media (prefers-color-scheme: dark) {
	:root {
		--bg: #0b1120;
		--surface: #131c31;
		--text: #e6edf6;
		--muted: #93a4bd;
		--border: #233049;
	}
	body { background: var(--bg); color: var(--text); }
}
body { background: var(--bg); color: var(--text); font-family: ui-sans-serif, system-ui, -apple-system, sans-serif; margin: 0; line-height: 1.6; }
* { box-sizing: border-box; }
code { font-family: ui-monospace, monospace; font-size: 0.88em; background: var(--border); padding: 0.1rem 0.35rem; }
`

const dashboardJS = `
(function() {
  var scriptSrc = document.currentScript.src;
  var path = scriptSrc.replace(/\/dashboard\.js$/, "");
  var base = path.replace(/\/-\/$/, "");
  var streamUrl = base + "/-/events/stream";
  var indicator = document.querySelector("[data-live-indicator]");
  var statusEl = document.querySelector("[data-sse-status]");
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

  function handleEvent(e) {
    try {
      var data = JSON.parse(e.data);
      document.dispatchEvent(new CustomEvent("dashboard:event", { detail: data }));
      eventCount++;
      updateStatus("open");
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
});`
