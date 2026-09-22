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

  // NOTE: the popover positioner that used to be vendored here (capture-phase
  // mirror of the library's algorithm) was removed — templ-components v1.19.0
  // ships the fix itself (capture-phase toggle listener + attach-time
  // self-heal for already-open panels). The library singleton
  // (window.tcPopoverPositionAttached) now owns anchor positioning.

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
  document.addEventListener(
    "close",
    function () {
      confirmIssue = null; // cancelled (button, backdrop, or Escape)
    },
    true,
  );
})();
