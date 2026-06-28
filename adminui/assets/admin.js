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
      setTimeout(function () {
        el.remove();
      }, 220);
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

  // --- Honest UI: sync-state lifecycle (ADR 0024) ---
  // Tracks pending mutations, listens for ACK confirmations over SSE,
  // and flips data-sync-state attributes on matching DOM elements.
  // Never-silent rollback: rejected items stay visible with error + retry.

  var sync = {
    pending: 0,
    confirmed: 0,
    failed: 0,
  };

  function updateIndicator() {
    var bar = document.querySelector("[data-sync-status]");
    if (!bar) return;

    var status, text;
    if (sync.failed > 0) {
      status = "failed";
      text = sync.failed + " failed — retry";
    } else if (sync.pending > 0) {
      status = "pending";
      text = sync.pending + " pending — syncing…";
    } else if (sync.confirmed > 0) {
      status = "ok";
      text = "All changes saved";
      // Auto-fade to idle after 2s
      setTimeout(function () {
        bar.setAttribute("data-sync-status", "idle");
        bar.textContent = "Synced";
        sync.confirmed = 0;
      }, 2000);
    } else {
      status = "idle";
      text = "Synced";
    }

    bar.setAttribute("data-sync-status", status);
    bar.textContent = text;
  }

  function setSyncState(element, state) {
    element.setAttribute("data-sync-state", state);
  }

  function handleSyncAck(detail) {
    if (!detail || !detail.commandId) return;

    var el = document.querySelector(
      '[data-command-id="' + detail.commandId + '"]'
    );
    if (!el) return;

    if (detail.status === "confirmed") {
      setSyncState(el, "confirmed");
      sync.pending = Math.max(0, sync.pending - 1);
      sync.confirmed++;
      announce(el, "Change saved");
    } else if (detail.status === "rejected") {
      setSyncState(el, "rejected");
      sync.pending = Math.max(0, sync.pending - 1);
      sync.failed++;
      if (detail.error) {
        announce(el, "Failed: " + detail.error, true);
      }
    }
    updateIndicator();
  }

  // aria-live region for screen reader announcements (confirmed only).
  // Created lazily on first use.
  var liveRegion = null;
  function announce(element, message, isError) {
    if (!liveRegion) {
      liveRegion = document.querySelector("[data-sync-live]");
      if (!liveRegion) {
        liveRegion = document.createElement("div");
        liveRegion.setAttribute("aria-live", "polite");
        liveRegion.setAttribute("aria-atomic", "true");
        liveRegion.setAttribute("data-sync-live", "");
        liveRegion.style.cssText =
          "position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0,0,0,0);";
        document.body.appendChild(liveRegion);
      }
    }
    // Only announce confirmed changes (not pending), per accessibility spec.
    if (!isError) {
      liveRegion.textContent = message;
    }
  }

  // --- SSE connection manager (auto-reconnect via EventSource) ---
  var eventSource = null;
  function connectSSE() {
    var sseURL = document.body.getAttribute("data-sse-url");
    if (!sseURL || typeof EventSource === "undefined") return;

    eventSource = new EventSource(sseURL);

    eventSource.addEventListener("sync:ack", function (e) {
      try {
        handleSyncAck(JSON.parse(e.data));
      } catch (err) {
        // Ignore malformed ACK payloads
      }
    });

    eventSource.addEventListener("open", function () {
      var bar = document.querySelector("[data-sync-status]");
      if (bar && sync.pending === 0) {
        bar.setAttribute("data-sync-status", "ok");
        bar.textContent = "Connected";
      }
    });

    eventSource.onerror = function () {
      // EventSource auto-reconnects; just update the indicator
      var bar = document.querySelector("[data-sync-status]");
      if (bar && sync.pending > 0) {
        bar.setAttribute("data-sync-status", "pending");
        bar.textContent = "Reconnecting…";
      }
    };
  }

  // --- Optimistic render: mark pending on htmx:beforeRequest ---
  // Auto-generates X-Command-Id for mutation requests (POST/PUT/DELETE)
  // so every destructive action is tracked without manual hx-headers.
  document.addEventListener("htmx:beforeRequest", function (e) {
    var verb = (e.detail.requestConfig.verb || "").toLowerCase();
    var isMutation = verb === "post" || verb === "put" || verb === "delete";
    if (!isMutation) return;

    e.detail.requestConfig.headers = e.detail.requestConfig.headers || {};
    var cmdID = e.detail.requestConfig.headers["X-Command-Id"];
    if (!cmdID && typeof crypto !== "undefined" && crypto.randomUUID) {
      cmdID = crypto.randomUUID();
      e.detail.requestConfig.headers["X-Command-Id"] = cmdID;
    }
    if (!cmdID) return;

    var target = e.detail.elt;
    // Walk up to find the closest element that should show sync-state
    var syncEl =
      target.closest("[data-sync-target]") ||
      target.closest("tr") ||
      target.closest("li") ||
      target;
    syncEl.setAttribute("data-command-id", cmdID);
    setSyncState(syncEl, "pending");
    sync.pending++;
    updateIndicator();
  });

  // --- Never-silent rollback: on transport error, show rejected ---
  document.addEventListener("htmx:responseError", function (e) {
    var target = e.detail.elt;
    var syncEl =
      target.closest("[data-command-id]") ||
      target.closest("[data-sync-state]");
    if (syncEl) {
      setSyncState(syncEl, "rejected");
      sync.pending = Math.max(0, sync.pending - 1);
      sync.failed++;
      announce(syncEl, "Network error — change not saved", true);
      updateIndicator();
    }
  });

  // --- Retry button: re-dispatch a rejected command ---
  document.addEventListener("click", function (e) {
    var btn = e.target.closest("[data-sync-retry]");
    if (!btn) return;

    var row = btn.closest("[data-command-id]");
    if (!row) return;

    // Clear rejected state and re-trigger via HTMX if the original element exists
    setSyncState(row, "pending");
    sync.failed = Math.max(0, sync.failed - 1);
    sync.pending++;
    updateIndicator();

    // If the row has an hx-post/hx-get, re-issue it
    var trigger = row.querySelector("[hx-post], [hx-get]");
    if (trigger && typeof htmx !== "undefined") {
      htmx.trigger(trigger, "retry");
    }
  });

  // --- Boot: connect SSE on DOMContentLoaded ---
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", connectSSE);
  } else {
    connectSSE();
  }
})();
