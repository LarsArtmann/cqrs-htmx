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
  document.addEventListener("keydown", function (e) {
    // "/" focuses the list search — unless the user is already typing.
    if (
      e.key === "/" &&
      !e.ctrlKey &&
      !e.metaKey &&
      !e.altKey &&
      !isTypingTarget(document.activeElement)
    ) {
      var search = listSearchInput();
      if (search) {
        e.preventDefault();
        search.focus();
      }
      return;
    }
    // Escape in the search clears it and re-runs the (htmx) search.
    if (
      e.key === "Escape" &&
      isTypingTarget(document.activeElement) &&
      document.activeElement.type === "search"
    ) {
      var box = document.activeElement;
      if (box.value !== "") {
        box.value = "";
        // Fire input so htmx's own trigger map (and any debounced listener)
        // sees the clear, then submit the wrapping form as the no-JS path.
        box.dispatchEvent(new Event("input", { bubbles: true }));
        var form = box.closest("form");
        if (form && typeof form.requestSubmit === "function") form.requestSubmit();
      }
      return;
    }
    // Arrow keys walk the list rows once focus is on a row or in the search.
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      var active = document.activeElement;
      var onRow = active && active.tagName === "TR";
      var inSearch = isTypingTarget(active) && active.type === "search";
      if (!onRow && !inSearch) return;
      var rows = listRows();
      if (rows.length === 0) return;
      e.preventDefault();
      var current = rows.indexOf(active);
      if (current === -1) {
        focusRow(rows, e.key === "ArrowDown" ? 0 : rows.length - 1);
        return;
      }
      focusRow(rows, e.key === "ArrowDown" ? current + 1 : current - 1);
    }
  });

  // --- Reduced motion (T16 F121): mirror the media query onto a class ---
  // prefers-reduced-motion is honored by the panel's own CSS (media query)
  // AND by this class on <html>, so a mid-session preference change takes
  // effect immediately and any future class-driven rule has a single hook.
  // classList only — no inline styles, CSP-safe. Toasts and spinners keep
  // their CONTENT (the library's motion-reduce: classes already strip their
  // animation); this guard covers the panel-owned animations (sync pulse).
  var motionQuery = window.matchMedia ? window.matchMedia("(prefers-reduced-motion: reduce)") : null;
  function applyMotionPreference() {
    document.documentElement.classList.toggle("admin-reduced-motion", !!(motionQuery && motionQuery.matches));
  }
  applyMotionPreference();
  if (motionQuery && typeof motionQuery.addEventListener === "function") {
    motionQuery.addEventListener("change", applyMotionPreference);
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
  // --- T22 F163: ownership-transfer confirm names BOTH sides ---
  // The select carries, per candidate, a server-composed confirm body naming
  // the prior owner and the receiver. On submit (capture, before htmx's own
  // htmx:confirm reads the attribute), the selected option's body is copied
  // onto the submit button's data-confirm — so the modal always describes
  // the exact pairing the admin picked, never a generic warning.
  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form.matches || !form.matches("form[data-transfer-confirm]")) return;
    var select = form.querySelector("select[name='email']");
    var button = form.querySelector("[data-confirm]");
    if (!select || !button) return;
    var opt = select.selectedOptions && select.selectedOptions[0];
    var body = opt && opt.getAttribute("data-confirm-body");
    if (body) button.setAttribute("data-confirm", body);
  }, true);

  // --- T21 F152: slug auto-suggest on the tenant-new form ---
  // Typing the display name suggests a slug (lowercase, non-alphanumerics →
  // dashes). The slug field stays freely editable; once the admin edits it
  // by hand (data-slug-touched), the suggest NEVER overwrites their text.
  // Document-delegated (gotcha 57): the form is server-rendered per page.
  function slugify(value) {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  }
  document.addEventListener("input", function (e) {
    var target = e.target;
    if (!target.matches || !target.matches("#display_name")) return;
    var form = target.closest("form");
    var slug = form && form.querySelector("[data-slug-suggest]");
    if (!slug || slug.getAttribute("data-slug-touched") === "true") return;
    slug.value = slugify(target.value);
  });
  document.addEventListener("input", function (e) {
    var t = e.target;
    if (t.matches && t.matches("[data-slug-suggest]")) {
      // Any direct edit in the slug field silences the suggester for good —
      // no forced overwrite, ever.
      t.setAttribute("data-slug-touched", "true");
    }
  });

  // --- T21 F158: tenant-switcher in-dropdown search ---
  // With many tenants the switcher dropdown gets a filter input, injected
  // the first time it opens (only when the list is long enough to need it)
  // and narrowing items by their data-tenant-search text. Client-side narrow
  // of the server-rendered list only — never a second source of truth.
  // The library renders the popover panel as "<ID>-menu" (dropdown_templ.go).
  var SWITCHER_PANEL_ID = "admin-tenant-switcher-menu";
  var SWITCHER_MIN_ITEMS = 8;
  function wireSwitcherSearch() {
    var panel = document.getElementById(SWITCHER_PANEL_ID);
    if (!panel || panel.dataset.switcherSearchWired === "true") return;
    var items = panel.querySelectorAll("[data-tenant-search]");
    if (items.length < SWITCHER_MIN_ITEMS) return;
    panel.dataset.switcherSearchWired = "true";
    var input = document.createElement("input");
    input.type = "search";
    input.placeholder = "Filter tenants…";
    input.setAttribute("aria-label", "Filter tenants");
    input.className = "mx-3 my-2 block w-[calc(100%-1.5rem)] rounded-md border border-gray-300 px-2 py-1 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100";
    input.addEventListener("input", function () {
      var needle = input.value.toLowerCase();
      for (var i = 0; i < items.length; i++) {
        var hit = items[i].getAttribute("data-tenant-search").indexOf(needle) !== -1;
        items[i].style.display = hit ? "" : "none";
      }
    });
    // Escape clears the filter (and closes via the popover's own light-dismiss).
    input.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && input.value !== "") {
        input.value = "";
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });
    panel.insertBefore(input, panel.firstChild);
  }
  document.addEventListener("toggle", function (e) {
    // toggle fires on popover elements when they open/close (popover API).
    if (e.target && e.target.id === SWITCHER_PANEL_ID && e.newState === "open") {
      wireSwitcherSearch();
    }
  }, true);
})();

// ==========================================================================
// T24 F175 — SSE-event-triggered refresh (LIVE UPDATES; append-only block).
// ==========================================================================
// The panel itself NEVER opens an EventSource: the SSE stream is the
// consumer's to wire (Config.SSEURL -> layout body data-sse-url -> the
// consumer's sync-client.js opens the connection). This block only REACTS:
// when the consumer's sync runtime announces events, the panel refreshes the
// dashboard stats region and any visible list region so a second operator's
// change shows up without a manual reload.
//
// Contract: sync/sync-client.js dispatches "cqrshtmx:sync-events" on
// `document` as a CustomEvent AND via htmx.trigger on <body> (which bubbles
// as the same DOM event). One document-level listener sees both, so this is
// CSP-safe (no inline handlers) and document-delegated (gotcha 57:
// element-bound JS dies with the pre-swap DOM).
(function () {
  "use strict";

  var REFRESH_SELECTOR = [
    // The polled stats region: found by its own hx-get tail (PolledRegion
    // renders no stable id, and the URL tail is fixed by the panel route).
    'div[hx-get$="/partials/stats"]',
    // Visible list regions with server-side partial twins (htmx.RenderPartial
    // serves the fragment when HX-Request is set).
    "#users-table",
    "#tenants-table",
    "#members-directory-table",
    "#audit-panel"
  ].join(",");

  // Only react when the consumer opted into live sync: data-sse-url exists
  // iff Config.SSEURL was set (layout_templ sseBodyAttrs). Unset -> this
  // block registers nothing and opens nothing.
  function liveUpdatesEnabled() {
    var body = document.body;
    return !!(body && body.hasAttribute("data-sse-url") && window.htmx);
  }

  // Refresh one region. Stats swap the region itself (outerHTML — the
  // response IS the PolledRegion element, so polling survives); lists swap
  // their server fragment into the container (innerHTML) using the CURRENT
  // location so search/filter/sort/page selections survive the refresh.
  function refreshRegion(el) {
    if (!el) return;
    var hxGet = el.getAttribute("hx-get");
    if (hxGet && /\/partials\/stats$/.test(hxGet)) {
      window.htmx.ajax("GET", hxGet, { target: el, swap: "outerHTML" });
      return;
    }
    window.htmx.ajax("GET", window.location.pathname + window.location.search, {
      target: el,
      swap: "innerHTML",
      headers: { "HX-Request": "true" }
    });
  }

  // Coalesce bursts: sync-client may deliver a pull batch followed by SSE
  // acks within milliseconds. One trailing refresh per quiet 300ms window.
  var refreshTimer = null;
  function scheduleRefresh() {
    if (!liveUpdatesEnabled() || refreshTimer) return;
    refreshTimer = window.setTimeout(function () {
      refreshTimer = null;
      var regions = document.querySelectorAll(REFRESH_SELECTOR);
      for (var i = 0; i < regions.length; i++) refreshRegion(regions[i]);
    }, 300);
  }

  document.addEventListener("cqrshtmx:sync-events", scheduleRefresh);
})();
