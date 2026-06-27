// adminui runtime — tiny vanilla-JS companion for the HTMX panel.
// No framework, no dependencies. Wired via go:embed.
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

  // --- Toasts: render messages pushed via the HX-Trigger header ---
  function toast(message, kind) {
    var host = document.querySelector(".toast-host");
    if (!host) {
      host = document.createElement("div");
      host.className = "toast-host";
      document.body.appendChild(host);
    }
    var el = document.createElement("div");
    el.className = "toast" + (kind ? " toast--" + kind : "");
    el.textContent = message;
    host.appendChild(el);
    setTimeout(function () {
      el.style.opacity = "0";
      el.style.transition = "opacity .2s";
      setTimeout(function () { el.remove(); }, 220);
    }, 3200);
  }
  document.body.addEventListener("htmx:afterSettle", function (e) {
    var triggers = e.detail && e.detail.requestConfig && e.detail.requestConfig.headers;
    // HTMX triggers are also dispatched as custom events; listen for ours.
  });
  document.addEventListener("adminui:toast", function (e) {
    var d = e.detail || {};
    toast(d.message || "", d.kind);
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
