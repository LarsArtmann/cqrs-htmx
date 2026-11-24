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

  // ===================== T17 client pack A (F124–F128) =====================
  // Everything below is document-delegated (gotcha 57) and CSP-safe. This
  // block is T17-owned territory: sortable headers, dirty-form guard,
  // copy-row-as-JSON, skeleton spend. Later tasks append AFTER this block,
  // still inside the IIFE.

  // --- F124/F125: the skeleton spends itself after the FIRST swap ---
  // users-skeleton is server-rendered hidden (.htmx-indicator) and shown by
  // htmx only while the first in-flight swap loads the rows. Once any swap
  // has landed, the window is loaded data — the node is removed so later
  // swaps never flash a skeleton for data the client already has.
  document.addEventListener("htmx:afterSwap", function () {
    var skeleton = document.getElementById("users-skeleton");
    if (skeleton) skeleton.remove();
  });

  // --- F126: client-side sortable headers WITHIN the loaded window ---
  // th[data-sort-key] headers carry a button; rows carry data-sort-<key>
  // attributes. Clicking sorts the CURRENT page in place (aria-sort on the
  // th + a text indicator) — no server round-trip. The URL-carried server
  // sort (T03) stays the default: any navigation, pager click, or swap
  // re-renders the header state from the URL.
  function clientSortValue(row, key) {
    return row.getAttribute("data-sort-" + key) || "";
  }
  function clientSortCompare(aRow, bRow, key, dir) {
    var a = clientSortValue(aRow, key);
    var b = clientSortValue(bRow, key);
    var aNum = parseFloat(a);
    var bNum = parseFloat(b);
    var cmp;
    if (!isNaN(aNum) && !isNaN(bNum) && a !== "" && b !== "") {
      cmp = aNum - bNum;
    } else {
      cmp = String(a).localeCompare(String(b));
    }
    return dir === "descending" ? -cmp : cmp;
  }
  document.addEventListener("click", function (e) {
    var btn = e.target.closest && e.target.closest("[data-sort-key]");
    if (!btn || !document.getElementById("users-table")) return;
    var th = btn.closest("th");
    var table = btn.closest("table");
    var tbody = table && table.querySelector("tbody");
    if (!th || !tbody) return;
    var key = btn.getAttribute("data-sort-key");
    var current = th.getAttribute("aria-sort");
    var next = current === "ascending" ? "descending" : "ascending";
    // One active sort at a time: clear every sortable header, then set ours.
    var headers = table.querySelectorAll("th[data-sort-key]");
    for (var i = 0; i < headers.length; i++) {
      headers[i].setAttribute("aria-sort", "none");
      var ind = headers[i].querySelector("[data-sort-indicator]");
      if (ind) ind.textContent = "";
    }
    th.setAttribute("aria-sort", next);
    var activeInd = btn.querySelector("[data-sort-indicator]");
    if (activeInd) activeInd.textContent = next === "ascending" ? "↑" : "↓";
    // Re-append in the new order (stable, in-DOM, no network).
    var rows = Array.prototype.slice.call(tbody.querySelectorAll("tr"));
    rows.sort(function (a, b) {
      return clientSortCompare(a, b, key, next);
    });
    for (var r = 0; r < rows.length; r++) tbody.appendChild(rows[r]);
  });

  // --- F127: dirty-form tracking + beforeunload guard ---
  // Forms marked data-dirty-guard (the detail page's display-name/email
  // edits) are checked STATELESSLY at unload time: a field is dirty only if
  // its live value differs from its defaultValue, so reverting by hand
  // disarms the guard and a clean form never nags. Only actual diffs guard.
  function adminFormIsDirty(form) {
    var fields = form.querySelectorAll("input, textarea, select");
    for (var i = 0; i < fields.length; i++) {
      var f = fields[i];
      if (f.type === "hidden" || f.type === "submit" || f.disabled) continue;
      if (f.value !== f.defaultValue) return true;
    }
    return false;
  }
  window.addEventListener("beforeunload", function (e) {
    var guarded = document.querySelectorAll("form[data-dirty-guard]");
    for (var i = 0; i < guarded.length; i++) {
      if (adminFormIsDirty(guarded[i])) {
        e.preventDefault();
        e.returnValue = ""; // Chrome requires returnValue to prompt
        return;
      }
    }
  });

  // --- F128: copy-row-as-JSON support action (user DETAIL page) ---
  // [data-copy-json] buttons carry the server-composed payload; the click
  // bridges to the async Clipboard API and reports BOTH outcomes honestly
  // through the existing adminui:toast bridge — success and explicit
  // denial (no clipboard API at all, or the write promise rejected).
  document.addEventListener("click", function (e) {
    var btn = e.target.closest && e.target.closest("[data-copy-json]");
    if (!btn) return;
    var payload = btn.getAttribute("data-copy-json") || "";
    function copyToast(kind, message) {
      document.dispatchEvent(new CustomEvent("adminui:toast", { detail: { kind: kind, message: message } }));
    }
    var okMsg = btn.getAttribute("data-copy-ok") || "Copied.";
    var errMsg = btn.getAttribute("data-copy-err") || "Clipboard denied — nothing was copied.";
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      navigator.clipboard.writeText(payload).then(
        function () {
          copyToast("ok", okMsg);
        },
        function () {
          copyToast("err", errMsg);
        },
      );
    } else {
      copyToast("err", errMsg);
    }
  });
  // =================== end T17 client pack A block ===================
})();

// ==========================================================================
// T19 F137/F141 — bulk select + actions (BULK block; append-only).
// ==========================================================================
// Contract: row checkboxes (input[data-bulk-row]) + one select-all
// (input[data-bulk-select-all]) drive a server-rendered bulk bar
// (#users-bulk-bar) that stays hidden until at least one row is checked.
// Everything here is document-delegated (gotcha 57: the form and the rows
// are re-rendered by every list swap, so element-bound JS would die) and
// CSP-safe (no inline handlers, no eval).
(function () {
  "use strict";

  function bulkForm() {
    return document.getElementById("users-bulk-form");
  }
  function bulkRows() {
    var form = bulkForm();
    if (!form) return [];
    return Array.prototype.slice.call(form.querySelectorAll("input[data-bulk-row]"));
  }
  function checkedRows() {
    return bulkRows().filter(function (cb) {
      return cb.checked;
    });
  }

  // One state pass per change: bar visibility, count text, select-all
  // checked/indeterminate, and the selection export link's ids param.
  function updateBulkState() {
    var form = bulkForm();
    if (!form) return;
    var rows = bulkRows();
    var checked = checkedRows();
    var bar = document.getElementById("users-bulk-bar");
    if (bar) bar.hidden = checked.length === 0;
    var count = document.getElementById("bulk-count");
    if (count) {
      count.textContent = checked.length === 0
        ? ""
        : checked.length + " of " + rows.length + " on this page selected";
    }
    var selectAll = form.querySelector("input[data-bulk-select-all]");
    if (selectAll) {
      selectAll.checked = checked.length > 0 && checked.length === rows.length;
      // Indeterminate ONLY when the selection is partial — the honest
      // three-state ("some, not all") the checkbox vocabulary exists for.
      selectAll.indeterminate = checked.length > 0 && checked.length < rows.length;
    }
    var exportLink = document.getElementById("bulk-export-csv");
    if (exportLink) {
      var base = exportLink.getAttribute("data-export-base") || exportLink.getAttribute("href");
      if (checked.length === 0) {
        exportLink.setAttribute("href", base);
      } else {
        var ids = checked.map(function (cb) {
          return cb.value;
        });
        exportLink.setAttribute("href", base + "?ids=" + encodeURIComponent(ids.join(",")));
      }
    }
  }

  document.addEventListener("change", function (e) {
    var t = e.target;
    if (!t || !t.matches) return;
    if (t.matches("input[data-bulk-row]")) updateBulkState();
  });

  // Select-all on CLICK (not change): setting every row programmatically
  // from the change handler would re-enter change events per row.
  document.addEventListener("click", function (e) {
    var all = e.target.closest && e.target.closest("input[data-bulk-select-all]");
    if (!all) return;
    var want = all.checked;
    bulkRows().forEach(function (cb) {
      cb.checked = want;
    });
    updateBulkState();
  });

  // F141: the escalation confirm NAMES THE COUNT. Composed at submit time
  // (capture, before htmx's htmx:confirm reads data-confirm) onto the FORM
  // — htmx's confirm elt for a form-submitted request is the form — using
  // the submitter's action value, so the dialog always describes the exact
  // action + selection it is about to run (the T22 transfer pattern).
  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form.matches || !form.matches("form[data-bulk-confirm]")) return;
    var checked = checkedRows();
    if (checked.length === 0) return; // server refusal panel handles honesty
    var submitter = e.submitter;
    var action = submitter && submitter.value ? submitter.value : "";
    var roleSelect = form.querySelector("select[name='roles']");
    var role = roleSelect && roleSelect.value ? roleSelect.value : "selected";
    var n = checked.length;
    var msg;
    if (action === "grant") {
      msg =
        "Grant the " + role + " role to " + n + " user" + (n === 1 ? "" : "s") +
        "? Each user's outcome is reported individually.";
    } else {
      msg =
        "Send the verification email to " + n + " user" + (n === 1 ? "" : "s") +
        "? Each send is reported individually.";
    }
    form.setAttribute("data-confirm", msg);
    form.setAttribute("data-confirm-title", "Confirm bulk action");
  }, true);

  // Keep the bar/state honest after any swap that re-rendered the rows
  // (a search or pager swap resets the selection server-side).
  document.addEventListener("htmx:afterSwap", function (e) {
    var target = e.detail && e.detail.target;
    if (target && (target.id === "users-table" || target.closest && target.closest("#users-table"))) {
      updateBulkState();
    }
  });

  // Initial pass (no-JS default: the bar stays hidden).
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", updateBulkState);
  } else {
    updateBulkState();
  }
})();

// ==========================================================================
// T18 F131–F134 — COMMAND PALETTE (append-only block).
// ==========================================================================
// The palette dialog (#admin-palette) is server-rendered EMPTY by layout.templ
// (paletteDialog): this block builds the link index at OPEN time from the
// page's own DOM — no server endpoint, never stale. Index contract:
//
//   data-palette-label + data-palette-href     explicit entries (future regions)
//   .admin-sidebar-scope nav a[href]           panel pages
//   #users-table / #tenants-table /
//   #members-directory-table tbody tr          the CURRENT page's rows
//   #audit-panel a[href]                       the audit filter chips
//
// Row entries reuse attributes earlier tasks already render (the first
// anchor per row + data-sort-email when present) — templates need no new
// markup for their rows to be jump targets.
//
// Everything is document-delegated (gotcha 57) and CSP-safe (no inline
// handlers; navigation via location.href on Enter). The fuzzy filter is a
// case-insensitive subsequence match scored by position, so "ada" hits
// "Ada Lovelace" and "jloe" hits "Jane Loe".
(function () {
  "use strict";

  var PALETTE_ID = "admin-palette";
  var PALETTE_INPUT_ID = "admin-palette-input";
  var PALETTE_LIST_ID = "admin-palette-list";

  // Fuzzy score: -1 = no match. Subsequence match; higher is better.
  function fuzzyScore(needle, haystack) {
    needle = needle.toLowerCase();
    haystack = haystack.toLowerCase();
    if (!needle) return 0;
    var score = 0;
    var idx = 0;
    var streak = 0;
    for (var i = 0; i < needle.length; i++) {
      var ch = needle[i];
      var found = haystack.indexOf(ch, idx);
      if (found === -1) return -1;
      streak = found === idx ? streak + 1 : 1;
      score += 10 - Math.min(found - idx, 9) + streak;
      idx = found + 1;
    }
    return score;
  }

  // One palette entry builder: label + href + group.
  function entry(label, href, group) {
    if (!label || !href) return null;
    return { label: label.trim(), href: href, group: group };
  }

  function rowEntries(container, group) {
    var out = [];
    if (!container) return out;
    var rows = container.querySelectorAll("tbody tr");
    for (var i = 0; i < rows.length; i++) {
      var row = rows[i];
      var a = row.querySelector("a[href]");
      if (!a) continue;
      var label = row.getAttribute("data-sort-email") || a.textContent;
      var item = entry(label, a.getAttribute("href"), group);
      if (item) out.push(item);
    }
    return out;
  }

  // Build the index fresh on every open (F132).
  function buildIndex() {
    var items = [];
    // Explicit extension contract first (future regions opt in with attrs).
    var explicit = document.querySelectorAll("[data-palette-label][data-palette-href]");
    for (var i = 0; i < explicit.length; i++) {
      var el = explicit[i];
      var item = entry(el.getAttribute("data-palette-label"), el.getAttribute("data-palette-href"), "More");
      if (item) items.push(item);
    }
    // Pages: the sidebar's nav links (de-duplicated by href).
    var seen = {};
    var nav = document.querySelectorAll(".admin-sidebar-scope nav a[href], .admin-sidebar-scope a[href]");
    for (var n = 0; n < nav.length; n++) {
      var href = nav[n].getAttribute("href");
      if (href && !seen[href]) {
        seen[href] = true;
        var navItem = entry(nav[n].textContent, href, "Pages");
        if (navItem) items.push(navItem);
      }
    }
    // Current page rows + audit filters.
    items = items.concat(
      rowEntries(document.getElementById("users-table"), "Users on this page"),
      rowEntries(document.getElementById("tenants-table"), "Tenants on this page"),
      rowEntries(document.getElementById("members-directory-table"), "Members on this page")
    );
    var audit = document.getElementById("audit-panel");
    if (audit) {
      var chips = audit.querySelectorAll("a[href]");
      for (var c = 0; c < chips.length; c++) {
        var chip = entry("Audit filter: " + chips[c].textContent, chips[c].getAttribute("href"), "Audit filters");
        if (chip) items.push(chip);
      }
    }
    return items;
  }

  // Render the filtered list into the dialog's listbox.
  var active = null;
  function setActive(dialog, opt) {
    if (active) active.removeAttribute("data-palette-active");
    active = opt;
    if (active) {
      active.setAttribute("data-palette-active", "true");
      dialog.querySelector("input").setAttribute("aria-activedescendant", active.id);
      if (typeof active.scrollIntoView === "function") active.scrollIntoView({ block: "nearest" });
    } else {
      dialog.querySelector("input").removeAttribute("aria-activedescendant");
    }
  }

  function renderResults(dialog, items) {
    var list = dialog.querySelector("[data-palette-list]");
    var empty = dialog.querySelector("[data-palette-empty]");
    // Drop previous results (keep the empty-state row for re-use).
    var stale = list.querySelectorAll("[data-palette-option]");
    for (var i = 0; i < stale.length; i++) stale[i].remove();
    var any = items.length > 0;
    if (empty) empty.setAttribute("data-palette-hidden", any ? "true" : "false");
    for (var k = 0; k < items.length; k++) {
      var opt = document.createElement("div");
      opt.id = "admin-palette-opt-" + k;
      opt.setAttribute("role", "option");
      opt.setAttribute("data-palette-option", "true");
      opt.setAttribute("data-palette-href", items[k].href);
      opt.tabIndex = -1;
      opt.className =
        "flex cursor-pointer items-center justify-between gap-3 rounded-md px-3 py-2 text-sm text-gray-900 hover:bg-gray-100 dark:text-white dark:hover:bg-gray-800";
      var label = document.createElement("span");
      label.textContent = items[k].label;
      var group = document.createElement("span");
      group.className = "text-xs text-gray-500 dark:text-gray-400";
      group.textContent = items[k].group;
      opt.appendChild(label);
      opt.appendChild(group);
      list.appendChild(opt);
    }
    setActive(dialog, list.querySelector("[data-palette-option]"));
  }

  function visibleOptions(dialog) {
    return Array.prototype.slice.call(dialog.querySelectorAll("[data-palette-option]"));
  }

  function openPalette(dialog) {
    if (!dialog || typeof dialog.showModal !== "function") return;
    var input = document.getElementById(PALETTE_INPUT_ID);
    renderResults(dialog, buildIndex());
    if (input) {
      input.value = "";
      input.focus();
    }
    dialog.showModal();
  }

  function jump(href) {
    var dlg = document.getElementById(PALETTE_ID);
    if (dlg) dlg.close();
    window.location.href = href;
  }

  document.addEventListener("keydown", function (e) {
    var dialog = document.getElementById(PALETTE_ID);
    if (!dialog) return;
    // F134: Ctrl/Cmd+K opens the palette from anywhere.
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      if (!dialog.open) openPalette(dialog);
      else dialog.close();
      return;
    }
    if (!dialog.open) return;
    var input = document.getElementById(PALETTE_INPUT_ID);
    // F134: "?" toggles the shortcut-help section (only when not typing text
    // that contains it — plain "?" keystroke).
    if (e.key === "?" && e.target === input) {
      e.preventDefault();
      var help = dialog.querySelector("[data-palette-help]");
      if (help) help.classList.toggle("hidden");
      return;
    }
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      var opts = visibleOptions(dialog);
      if (opts.length === 0) return;
      e.preventDefault();
      var current = opts.indexOf(active);
      var next;
      if (current === -1) next = e.key === "ArrowDown" ? 0 : opts.length - 1;
      else next = e.key === "ArrowDown" ? Math.min(current + 1, opts.length - 1) : Math.max(current - 1, 0);
      setActive(dialog, opts[next]);
      return;
    }
    // F133: Enter jumps to the active result.
    if (e.key === "Enter") {
      e.preventDefault();
      if (active) jump(active.getAttribute("data-palette-href"));
    }
  });

  document.addEventListener("input", function (e) {
    var t = e.target;
    if (!t.matches || !t.matches("[data-palette-input]")) return;
    var dialog = document.getElementById(PALETTE_ID);
    if (!dialog) return;
    var items = buildIndex();
    var q = t.value.trim();
    if (q !== "") {
      var scored = [];
      for (var i = 0; i < items.length; i++) {
        var s = fuzzyScore(q, items[i].label);
        if (s >= 0) scored.push({ score: s, item: items[i] });
      }
      scored.sort(function (a, b) {
        return b.score - a.score;
      });
      items = [];
      for (var k = 0; k < scored.length; k++) items.push(scored[k].item);
    }
    renderResults(dialog, items);
  });

  // Click (and the backdrop, which the browser reports as a click on the
  // dialog itself) + close handling. Enter-jump and click-select converge on
  // jump(); closing the dialog clears the active option so a reopen starts
  // fresh.
  document.addEventListener("click", function (e) {
    var dialog = document.getElementById(PALETTE_ID);
    if (!dialog) return;
    if (e.target === dialog) {
      dialog.close();
      return;
    }
    var opt = e.target.closest && e.target.closest("[data-palette-option]");
    if (opt) jump(opt.getAttribute("data-palette-href"));
  });
  document.addEventListener("close", function (e) {
    if (e.target && e.target.id === PALETTE_ID) setActive(e.target, null);
  }, true);
})();
// ============================ end T18 palette block ============================

// ==========================================================================
// T20 F146–F149 — SAVED VIEWS + DISPLAY PREFS (append-only block).
// ==========================================================================
// Client-only persistence for the users list: no server endpoint, no server
// state. One namespaced, versioned localStorage key holds everything:
//
//   key:    "cqrshtmx:adminui:users-prefs:v1"
//   value:  {"version":1,"presets":[{"name":...,"query":"q=..&filter=.."}],
//            "hidden":["access","lastseen"],"density":"compact"}
//
// Preset contract (F147): a preset serializes the list URL query string
// (q/filter/sort/page — window.location.search at save time). Apply
// navigates to the list base URL + "?" + query (navigation IS the refresh;
// the server URL builder remains the single source of truth for what a view
// means). Column visibility (F148) toggles the classes
// admin-cols-hide-access / admin-cols-hide-lastseen on the #users-table
// container — the stylesheet hides the matching column POSITIONALLY; the
// table markup is never touched (it belongs to the concurrent rows agent).
// Density (F149) toggles admin-density-compact on the same container.
(function () {
  "use strict";

  var STORE_KEY = "cqrshtmx:adminui:users-prefs:v1";
  var STORE_VERSION = 1;
  var TABLE_ID = "users-table";
  var COL_CLASSES = { access: "admin-cols-hide-access", lastseen: "admin-cols-hide-lastseen" };

  function readStore() {
    try {
      var raw = window.localStorage.getItem(STORE_KEY);
      if (!raw) return { version: STORE_VERSION, presets: [], hidden: [], density: "comfortable" };
      var parsed = JSON.parse(raw);
      if (!parsed || parsed.version !== STORE_VERSION) {
        return { version: STORE_VERSION, presets: [], hidden: [], density: "comfortable" };
      }
      return parsed;
    } catch (err) {
      // Corrupt or blocked storage degrades to defaults — never to a crash.
      return { version: STORE_VERSION, presets: [], hidden: [], density: "comfortable" };
    }
  }

  function writeStore(store) {
    try {
      window.localStorage.setItem(STORE_KEY, JSON.stringify(store));
    } catch (err) {
      // Storage full/blocked: the panel keeps working, persistence is lost.
    }
  }

  function applyPrefs(store) {
    var table = document.getElementById(TABLE_ID);
    if (!table) return;
    for (var col in COL_CLASSES) {
      if (Object.prototype.hasOwnProperty.call(COL_CLASSES, col)) {
        table.classList.toggle(COL_CLASSES[col], store.hidden.indexOf(col) !== -1);
        var btn = document.querySelector('[data-prefs-col="' + col + '"]');
        if (btn) btn.setAttribute("aria-pressed", store.hidden.indexOf(col) !== -1 ? "true" : "false");
      }
    }
    var compact = store.density === "compact";
    table.classList.toggle("admin-density-compact", compact);
    var dens = document.querySelector("[data-prefs-density]");
    if (dens) {
      dens.setAttribute("aria-pressed", compact ? "true" : "false");
      dens.textContent = compact ? "Comfortable" : "Compact";
    }
  }

  function renderPresets(store) {
    var host = document.querySelector("[data-prefs-presets]");
    if (!host) return;
    while (host.firstChild) host.removeChild(host.firstChild);
    if (store.presets.length === 0) {
      var none = document.createElement("span");
      none.className = "text-gray-600 dark:text-gray-400";
      none.setAttribute("data-prefs-empty", "true");
      none.textContent = "No saved views yet.";
      host.appendChild(none);
      return;
    }
    for (var i = 0; i < store.presets.length; i++) {
      (function (preset) {
        var apply = document.createElement("button");
        apply.type = "button";
        apply.setAttribute("data-prefs-apply", preset.name);
        apply.setAttribute(
          "class",
          "rounded-full bg-gray-900 px-2.5 py-0.5 font-medium text-white dark:bg-white dark:text-gray-900"
        );
        apply.textContent = preset.name;
        apply.title = "Apply view: " + preset.query;
        var del = document.createElement("button");
        del.type = "button";
        del.setAttribute("data-prefs-delete", preset.name);
        del.setAttribute(
          "aria-label",
          "Delete view " + preset.name
        );
        del.setAttribute(
          "class",
          "rounded-full border border-gray-300 px-1.5 py-0.5 text-gray-500 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
        );
        del.textContent = "×";
        host.appendChild(apply);
        host.appendChild(del);
      })(store.presets[i]);
    }
  }

  function currentQuery() {
    // The preset IS the URL query (q/filter/sort/page) — nothing else.
    return window.location.search.replace(/^\?/, "");
  }

  function applyPreset(base, query) {
    window.location.href = base + (query ? "?" + query : "");
  }

  document.addEventListener("DOMContentLoaded", function () {
    var bar = document.querySelector("[data-users-prefs]");
    if (!bar) return;
    var store = readStore();
    applyPrefs(store);
    renderPresets(store);
  });

  // Save: name from the input; empty name ignored honestly.
  document.addEventListener("click", function (e) {
    var save = e.target.closest && e.target.closest("[data-prefs-save]");
    if (save) {
      var nameInput = document.querySelector("[data-prefs-name]");
      var name = nameInput ? nameInput.value.trim() : "";
      if (!name) return;
      var store = readStore();
      var query = currentQuery();
      var replaced = false;
      for (var i = 0; i < store.presets.length; i++) {
        if (store.presets[i].name === name) {
          store.presets[i].query = query;
          replaced = true;
        }
      }
      if (!replaced) store.presets.push({ name: name, query: query });
      writeStore(store);
      renderPresets(store);
      if (nameInput) nameInput.value = "";
      return;
    }
    var del = e.target.closest && e.target.closest("[data-prefs-delete]");
    if (del) {
      var name2 = del.getAttribute("data-prefs-delete");
      var store2 = readStore();
      store2.presets = store2.presets.filter(function (p) {
        return p.name !== name2;
      });
      writeStore(store2);
      renderPresets(store2);
      return;
    }
    var apply = e.target.closest && e.target.closest("[data-prefs-apply]");
    if (apply) {
      var bar = apply.closest("[data-users-prefs]");
      var base = bar ? bar.getAttribute("data-prefs-base") : "/users";
      var name3 = apply.getAttribute("data-prefs-apply");
      var store3 = readStore();
      for (var k = 0; k < store3.presets.length; k++) {
        if (store3.presets[k].name === name3) {
          applyPreset(base, store3.presets[k].query);
          return;
        }
      }
      return;
    }
    // Column visibility (F148): toggle the container class, persist.
    var colBtn = e.target.closest && e.target.closest("[data-prefs-col]");
    if (colBtn) {
      var col = colBtn.getAttribute("data-prefs-col");
      var store4 = readStore();
      var idx = store4.hidden.indexOf(col);
      if (idx === -1) store4.hidden.push(col);
      else store4.hidden.splice(idx, 1);
      writeStore(store4);
      applyPrefs(store4);
      return;
    }
    // Density (F149).
    var densBtn = e.target.closest && e.target.closest("[data-prefs-density]");
    if (densBtn) {
      var store5 = readStore();
      store5.density = store5.density === "compact" ? "comfortable" : "compact";
      writeStore(store5);
      applyPrefs(store5);
    }
  });
})();
// ============================ end T20 prefs block ============================
