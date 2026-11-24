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

  // --- Focus restore after HTMX swaps (T15 F110) ---
  // A swap that replaces the focused element (e.g. the audit date form is
  // inside #audit-panel, so "Apply dates" swaps away its own trigger) drops
  // keyboard focus to <body> — a hard stop for keyboard users mid-flow.
  // Both listeners are DOCUMENT-delegated and re-query per event (gotcha
  // 57: element-bound JS dies with the pre-swap DOM under hx-boost, and
  // synthetic htmx events must bubble to be seen here).
  //
  // Restore contract: templates mark restorable triggers with
  // data-focus-restore="<id>" (audit "Apply dates" button, pager arrows and
  // page links). After the swap, admin.js re-queries the FRESH DOM for the
  // same value — the pre-swap element reference is dead if its region
  // swapped, and element-bound handlers would be gone with it.
  var focusRestoreValue = null;
  document.addEventListener("htmx:beforeSwap", function (e) {
    focusRestoreValue = null;
    var elt = e.detail && e.detail.requestConfig && e.detail.requestConfig.elt;
    var trigger = elt && elt.closest ? elt.closest("[data-focus-restore]") : null;
    if (!trigger && document.activeElement && document.activeElement.closest) {
      trigger = document.activeElement.closest("[data-focus-restore]");
    }
    if (trigger) focusRestoreValue = trigger.getAttribute("data-focus-restore") || null;
  });
  document.addEventListener("htmx:afterSwap", function () {
    if (!focusRestoreValue) return;
    var want = focusRestoreValue;
    focusRestoreValue = null;
    var prior = document.querySelector('[data-focus-restore="' + want + '"]');
    if (prior && typeof prior.focus === "function") prior.focus();
  });

  // --- Keyboard list navigation (T15 F111) ---
  // "/" focuses the list search, Escape clears it (and re-runs the search),
  // ArrowDown/ArrowUp traverse the visible list rows. All document-delegated
  // (gotcha 57), CSP-safe, and inert while typing in non-search controls.
  function isTypingTarget(el) {
    if (!el || !el.tagName) return false;
    var tag = el.tagName.toLowerCase();
    return tag === "input" || tag === "textarea" || tag === "select" || el.isContentEditable;
  }
  function listSearchInput() {
    // The FilterInput search box on the list pages (users/tenants/audit):
    // re-queried per keypress so partial swaps never stale-cache it.
    return (
      document.querySelector("main input[type='search'], #panel input[type='search']") ||
      document.querySelector("input[type='search']")
    );
  }
  function listRowContainer() {
    var ids = ["users-table", "tenants-table", "audit-panel"];
    for (var i = 0; i < ids.length; i++) {
      var el = document.getElementById(ids[i]);
      if (el) return el;
    }
    return null;
  }
  function listRows() {
    var container = listRowContainer();
    if (!container) return [];
    var rows = container.querySelectorAll("tbody tr");
    var visible = [];
    for (var i = 0; i < rows.length; i++) {
      if (rows[i].offsetParent !== null) visible.push(rows[i]);
    }
    return visible;
  }
  function focusRow(rows, index) {
    if (index < 0 || index >= rows.length) return;
    var row = rows[index];
    // Rows are programmatic focus targets only (never in the Tab ring).
    if (!row.hasAttribute("tabindex")) row.setAttribute("tabindex", "-1");
    row.focus();
  }
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
