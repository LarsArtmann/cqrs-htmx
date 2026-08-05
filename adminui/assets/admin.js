// adminui runtime — tiny vanilla-JS companion for the HTMX panel.
// No framework, no dependencies. Wired via go:embed.
//
// Offline command sync (SSE + SharedWorker + IndexedDB) has been extracted
// to sync-client.js (served by the root cqrshtmx module). Include it via
// a <script> tag before this one if offline sync is desired.
(function () {
  "use strict";

  // --- CSRF: send token on every HTMX request (double-submit pattern) ---
  var meta = document.querySelector('meta[name="csrf-token"]');
  if (meta && typeof htmx !== "undefined") {
    htmx.config.headers = htmx.config.headers || {};
    htmx.config.headers["X-CSRF-Token"] = meta.content;
  }

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

  // --- Confirm before destructive actions ---
  document.addEventListener("htmx:confirm", function (e) {
    var elt = e.detail.elt;
    var msg = elt.getAttribute("data-confirm");
    if (msg) {
      e.preventDefault();
      if (window.confirm(msg)) e.detail.issueRequest(true);
    }
  });
})();
