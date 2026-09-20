// adminui runtime — tiny vanilla-JS companion for the HTMX panel.
// No framework, no dependencies. Wired via go:embed.
//
// Offline command sync (SSE + SharedWorker + IndexedDB) has been extracted
// to sync-client.js (served by the root cqrshtmx module). Include it via
// a <script> tag before this one if offline sync is desired.
(function () {
  "use strict";

  // --- CSRF: send token on every HTMX request (double-submit pattern) ---
  // htmx has no htmx.config.headers option (assigning one is a silent no-op —
  // button POSTs 403'd under CSRF middleware). The sanctioned hook is the
  // htmx:configRequest event, which carries the per-request header map.
  document.addEventListener("htmx:configRequest", function (e) {
    var meta = document.querySelector('meta[name="csrf-token"]');
    if (meta) e.detail.headers["X-CSRF-Token"] = meta.content;
  });

  // --- Vendored popover positioner (templ-components v1.18.1 upstream bug) ---
  // The library's anchor-positioning singleton registers on `toggle` WITHOUT
  // capture; the popover toggle event does not bubble, so that listener never
  // fires and Dropdown/Popover panels open at the viewport's top-left corner
  // instead of at their trigger. This capture-phase mirror of the library's
  // positioning (anchor rect + data-tc-position/data-tc-align, gap 8, viewport
  // clamping) stays until the upstream fix ships.
  function positionPopover(p) {
    var aid = p.getAttribute("data-tc-anchor");
    var t = aid ? document.getElementById(aid) : null;
    if (!t) return;
    var pos = p.getAttribute("data-tc-position") || "bottom";
    var align = p.getAttribute("data-tc-align") || "center";
    var gap = 8;
    var r = t.getBoundingClientRect();
    var w = p.offsetWidth, h = p.offsetHeight;
    var vw = window.innerWidth, vh = window.innerHeight;
    var left, top;
    if (align === "start") left = r.left;
    else if (align === "end") left = r.right - w;
    else left = r.left + r.width / 2 - w / 2;
    if (pos === "top") top = r.top - h - gap;
    else if (pos === "left") { top = r.top + r.height / 2 - h / 2; left = r.left - w - gap; }
    else if (pos === "right") { top = r.top + r.height / 2 - h / 2; left = r.right + gap; }
    else top = r.bottom + gap;
    if (pos === "bottom" && top + h > vh - gap && r.top - gap - h >= gap) top = r.top - h - gap;
    left = Math.max(gap, Math.min(left, vw - w - gap));
    top = Math.max(gap, Math.min(top, vh - h - gap));
    p.style.inset = "auto";
    p.style.left = left + "px";
    p.style.top = top + "px";
    p.style.margin = "0";
  }
  document.addEventListener("toggle", function (e) {
    var el = e.target;
    if (el && el.hasAttribute && el.hasAttribute("popover") && el.getAttribute("data-tc-anchor") && e.newState === "open") {
      positionPopover(el);
    }
  }, true);
  window.addEventListener("resize", function () {
    document.querySelectorAll("[popover]:popover-open[data-tc-anchor]").forEach(positionPopover);
  });
  document.addEventListener("scroll", function () {
    document.querySelectorAll("[popover]:popover-open[data-tc-anchor]").forEach(positionPopover);
  }, true);

  // --- Mobile sidebar toggle ---
  function toggleSidebar() {
    var sb = document.querySelector(".admin-sidebar");
    var sc = document.querySelector(".admin-scrim");
    if (!sb) return;
    var open = sb.classList.toggle("open");
    if (sc) sc.classList.toggle("open", open);
  }
  document.addEventListener("click", function (e) {
    var t = e.target.closest && e.target.closest(".admin-toggle");
    if (t) toggleSidebar();
    if (e.target.classList && e.target.classList.contains("admin-scrim")) toggleSidebar();
  });

  // --- Toasts: bridge adminui:toast HX-Trigger events to templ-components tcShowToast ---
  // tcShowToast is provided by feedback.ToastContainer (rendered server-side).
  document.addEventListener("adminui:toast", function (e) {
    var d = e.detail || {};
    var kindMap = {
      ok: "success",
      err: "error",
      warn: "warning",
      info: "info",
    };
    if (typeof tcShowToast === "function") {
      tcShowToast(d.message || "", kindMap[d.kind] || "info");
    } else {
      console.warn("adminui: tcShowToast not available — toast lost:", d.message);
    }
  });

  // --- Confirm destructive actions via the shared native <dialog> modal ---
  // Buttons carry data-confirm (body) + optional data-confirm-title. The
  // #admin-confirm-modal dialog is rendered once by the server (Layout
  // pageFoot). Confirm issues the intercepted htmx request; Cancel, backdrop
  // click, and Escape (all native dialog behavior) drop it. Falls back to
  // window.confirm when the dialog is missing or <dialog> unsupported.
  var confirmIssue = null;
  document.addEventListener("htmx:confirm", function (e) {
    var elt = e.detail.elt;
    var msg = elt.getAttribute("data-confirm");
    if (!msg) return;
    e.preventDefault();
    var dlg = document.getElementById("admin-confirm-modal");
    if (!dlg || typeof dlg.showModal !== "function") {
      if (window.confirm(msg)) e.detail.issueRequest(true);
      return;
    }
    confirmIssue = function () {
      e.detail.issueRequest(true);
    };
    var body = document.getElementById("admin-confirm-body");
    var title = document.getElementById("admin-confirm-modal-title");
    if (body) body.textContent = msg;
    if (title) {
      title.textContent = elt.getAttribute("data-confirm-title") || "Are you sure?";
    }
    dlg.showModal();
  });
  document.addEventListener("click", function (e) {
    var ok = e.target.closest && e.target.closest("#admin-confirm-ok");
    if (!ok || !confirmIssue) return;
    var issue = confirmIssue;
    confirmIssue = null;
    var dlg = document.getElementById("admin-confirm-modal");
    if (dlg) dlg.close();
    issue();
  });
  document.addEventListener("close", function () {
    confirmIssue = null; // cancelled (button, backdrop, or Escape)
  }, true);
})();
