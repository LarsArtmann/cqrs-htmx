// sync-client.js — Tab-side offline command sync client (ADR 0024 + 0029 + 0040).
//
// Self-contained vanilla-JS client for the cqrs-htmx offline sync stack.
// Works with any HTMX frontend — no admin UI required.
//
// WHAT IT DOES:
//   1. Listens to HTMX events (beforeRequest, sendError, responseError) to
//      track mutations and auto-stamp X-Command-Id headers.
//   2. Connects SSE for server ACK confirmations (sync:ack events).
//   3. Coordinates with a SharedWorker (sync-worker.js) for offline command
//      queueing with IndexedDB persistence.
//   4. Manages a sync indicator element ([data-sync-status]) showing
//      pending/confirmed/failed/queued counts.
//   5. Retries queued commands on reconnect via htmx.trigger() or
//      htmx.ajax() for cross-session recovery.
//
// ACTIVATION:
//   The client auto-initializes on DOMContentLoaded if the <body> element
//   has a [data-sse-url] attribute. No data-sse-url = no sync (graceful no-op).
//
// SHAREDWORKER URL:
//   By default, derived from this script's own <script src> path: it replaces
//   "sync-client.js" with "sync-worker.js" in the URL. Both must be served
//   under the same base path.
//   Override with a data-sync-worker-url attribute on the <script> tag if
//   the worker is mounted at a different path:
//     <script src="/assets/client.js" data-sync-worker-url="/workers/sync.js">
//
// NO BUILD STEP. No framework. No dependencies beyond HTMX (loaded separately).
// @ts-check
"use strict";

(function () {
  const VERSION = "1.2.0";

  // --- Sync state: tracks pending/confirmed/failed/queued mutation counts ---
  const sync = {
    pending: 0,
    confirmed: 0,
    failed: 0,
    queued: 0,
  };

  /**
   * Update the [data-sync-status] indicator element with current sync state.
   * Called whenever pending/confirmed/failed/queued counts change.
   */
  function updateIndicator() {
    const bar = document.querySelector("[data-sync-status]");
    if (!bar) return;

    let status, text;
    if (sync.queued > 0) {
      status = "offline";
      text = sync.queued + " queued — offline";
    } else if (sync.failed > 0) {
      status = "failed";
      text = sync.failed + " failed — retry";
    } else if (sync.pending > 0) {
      status = "pending";
      text = sync.pending + " pending — syncing…";
    } else if (sync.confirmed > 0) {
      status = "ok";
      text = "All changes saved";
      // Auto-fade to idle after 2s
      setTimeout(() => {
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

  // --- aria-live region for screen reader announcements (confirmed only) ---
  let liveRegion = null;
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
    if (!isError) {
      liveRegion.textContent = message;
    }
  }

  // --- SSE connection manager (auto-reconnect via EventSource) ---
  let eventSource = null;
  /**
   * Connect to the SSE endpoint for server ACK confirmations.
   * Reads the URL from <body data-sse-url>. Auto-reconnects via EventSource.
   * No-op if no data-sse-url or EventSource unavailable.
   */
  function connectSSE() {
    const sseURL = document.body.getAttribute("data-sse-url");
    if (!sseURL || typeof EventSource === "undefined") return;

    eventSource = new EventSource(sseURL);

    eventSource.addEventListener("sync:ack", (e) => {
      try {
        handleSyncAck(JSON.parse(e.data));
      } catch (err) {
        // Ignore malformed ACK payloads
      }
    });

    eventSource.addEventListener("open", () => {
      const bar = document.querySelector("[data-sync-status]");
      if (bar && sync.pending === 0) {
        bar.setAttribute("data-sync-status", "ok");
        bar.textContent = "Connected";
      }
    });

    eventSource.onerror = () => {
      // EventSource auto-reconnects; just update the indicator
      const bar = document.querySelector("[data-sync-status]");
      if (bar && sync.pending > 0) {
        bar.setAttribute("data-sync-status", "pending");
        bar.textContent = "Reconnecting…";
      }
    };
  }

  // --- Sync ACK handler: flips DOM state on server confirmation/rejection ---
  /**
   * Handle a server ACK (confirmed or rejected) for a tracked command.
   * Flips the DOM element's sync state and updates counters.
   * @param {{ commandId: string, status: string, error?: string }} detail - ACK payload from SSE.
   */
  function handleSyncAck(detail) {
    if (!detail || !detail.commandId) return;

    const el = document.querySelector('[data-command-id="' + detail.commandId + '"]');
    if (!el) return;

    if (detail.status === "confirmed") {
      setSyncState(el, "confirmed");
      sync.pending = Math.max(0, sync.pending - 1);
      sync.confirmed++;
      announce(el, "Change saved");
      ackCommand(detail.commandId);
    } else if (detail.status === "rejected") {
      setSyncState(el, "rejected");
      sync.pending = Math.max(0, sync.pending - 1);
      sync.failed++;
      if (detail.error) {
        announce(el, "Failed: " + detail.error, true);
      }
      ackCommand(detail.commandId);
    }
    updateIndicator();
  }

  // --- Offline command queue (ADR 0029 + ADR 0040): SharedWorker coordination ---
  let syncWorker = null;
  let tabId = null;

  /**
   * Initialize the SharedWorker connection. Derives the worker URL from
   * this script's <script src> path (or data-sync-worker-url override),
   * registers a tabId, and wires up retry/pending/dead message handlers.
   * No-op if SharedWorker is unavailable (graceful degradation).
   */
  function initSyncWorker() {
    if (typeof SharedWorker === "undefined") return;

    // Find this script's tag to derive the worker URL.
    const script = document.querySelector('script[src$="sync-client.js"]');
    if (!script) return;

    // Allow consumers to override the worker URL via a data attribute
    // when the worker is mounted at a different path than the client.
    let workerURL = script.getAttribute("data-sync-worker-url");
    if (!workerURL) {
      const basePath = script.src.replace(/\/sync-client\.js$/, "");
      workerURL = basePath + "/sync-worker.js";
    }

    try {
      syncWorker = new SharedWorker(workerURL);
      tabId =
        typeof crypto !== "undefined" && crypto.randomUUID
          ? crypto.randomUUID()
          : String(Date.now()) + Math.random().toString(36).slice(2);

      syncWorker.port.onmessage = (e) => {
        const data = e.data;
        if (!data || !data.type) return;

        if (data.type === "retry") {
          retryQueuedCommand(data.commandId, data.envelope);
        } else if (data.type === "pending") {
          sync.queued = Math.max(sync.queued, data.count | 0);
          updateIndicator();
        } else if (data.type === "dead") {
          handleDeadCommand(data.commandId);
        }
      };
      syncWorker.port.start();

      // Register this tab with the worker so it can target retry messages
      // to the originating tab and clean up on disconnect.
      syncWorker.port.postMessage({ type: "hello", tabId: tabId });

      // Best-effort unregister on beforeunload (tab close, navigation).
      window.addEventListener("beforeunload", () => {
        if (syncWorker) {
          try {
            syncWorker.port.postMessage({ type: "bye", tabId: tabId });
          } catch (e) {
            // Worker already gone — nothing to clean up
          }
        }
      });

      // When the page detects connectivity return, tell the worker to flush.
      // The window online/offline events fire reliably in the page context
      // (unlike in the SharedWorker scope, which may not receive them in
      // some browsers or test environments like Playwright).
      window.addEventListener("online", () => {
        if (syncWorker) {
          try {
            syncWorker.port.postMessage({ type: "flush" });
          } catch (e) {
            // Worker gone — nothing to flush
          }
        }
      });
    } catch (e) {
      // SharedWorker unavailable — online path unaffected (graceful degradation)
    }
  }

  /**
   * Queue a command envelope to the SharedWorker for offline persistence.
   * @param {string} commandId - Unique command identifier.
   * @param {{ verb: string, url: string, values: Object|null, headers: Object|null }|null} envelope - Request data for retry.
   */
  function enqueueCommand(commandId, envelope) {
    if (!syncWorker || !commandId) return;
    syncWorker.port.postMessage({
      type: "enqueue",
      commandId: commandId,
      envelope: envelope || null,
    });
    sync.queued++;
    updateIndicator();
  }

  function ackCommand(commandId) {
    if (!syncWorker || !commandId) return;
    syncWorker.port.postMessage({ type: "ack", commandId: commandId });
  }

  // handleDeadCommand: the worker gave up after MAX_RETRIES or TTL.
  function handleDeadCommand(commandId) {
    if (!commandId) return;
    const el = document.querySelector('[data-command-id="' + commandId + '"]');
    if (el) {
      setSyncState(el, "rejected");
      announce(el, "Sync failed after retries — manual retry needed", true);
    }
    sync.queued = Math.max(0, sync.queued - 1);
    sync.failed++;
    updateIndicator();
  }

  /**
   * Retry a queued command. If the originating DOM element still exists,
   * re-trigger it via htmx.trigger(). If gone, use rebuildAndRetry() to
   * synthesize a new host element and re-issue via htmx.ajax().
   * @param {string} commandId - Unique command identifier.
   * @param {{ verb: string, url: string, values: Object|null, headers: Object|null }} [envelope] - Persisted request data for cross-session retry.
   */
  function retryQueuedCommand(commandId, envelope) {
    if (!commandId) return;
    const selector = '[data-command-id="' + commandId + '"]';
    const el = document.querySelector(selector);
    if (!el) {
      // Element gone (user navigated away). If we have a persisted envelope
      // (ADR-0040 cross-tab/cross-session retry), rebuild the request via the
      // HTMX JS API into a fresh row so the command is not silently lost.
      if (envelope && typeof htmx !== "undefined" && htmx.ajax) {
        rebuildAndRetry(commandId, envelope);
        return;
      }
      // No envelope and no element — honest: show as failed, not silent.
      sync.queued = Math.max(0, sync.queued - 1);
      sync.failed++;
      updateIndicator();
      ackCommand(commandId);
      return;
    }
    // Clear queued state, transition to pending (re-flight)
    el.removeAttribute("data-sync-queued");
    sync.queued = Math.max(0, sync.queued - 1);
    setSyncState(el, "pending");
    sync.pending++;
    updateIndicator();
    // Re-issue via htmx.ajax when we have a persisted envelope. This is more
    // reliable than htmx.trigger(el, "click"), which only works if el itself
    // has an HTMX trigger attribute (forms trigger on submit, not click).
    if (envelope && typeof htmx !== "undefined" && htmx.ajax) {
      htmx.ajax(envelope.verb || "POST", envelope.url, {
        target: el,
        swap: "outerHTML",
        values: envelope.values || null,
        headers: envelope.headers || null,
      });
    } else if (typeof htmx !== "undefined") {
      htmx.trigger(el, "click");
    }
  }

  /**
   * Rebuild a DOM element for a persisted command whose originating element
   * is gone (cross-session retry after browser restart). Uses htmx.ajax()
   * to re-issue the request into a fresh host element.
   * @param {string} commandId - Unique command identifier.
   * @param {{ verb: string, url: string, values: Object|null, headers: Object|null }} envelope - Persisted request data.
   */
  function rebuildAndRetry(commandId, envelope) {
    const host = document.createElement("div");
    host.setAttribute("data-command-id", commandId);
    host.setAttribute("data-sync-state", "pending");
    document.body.appendChild(host);
    sync.queued = Math.max(0, sync.queued - 1);
    sync.pending++;
    updateIndicator();
    htmx.ajax(envelope.verb || "POST", envelope.url, {
      target: host,
      swap: "outerHTML",
      values: envelope.values || null,
      headers: envelope.headers || null,
    });
  }

  // --- Optimistic render: mark pending on htmx:beforeRequest ---
  // Auto-generates X-Command-Id for mutation requests (POST/PUT/DELETE)
  // so every destructive action is tracked without manual hx-headers.
  document.addEventListener("htmx:beforeRequest", (e) => {
    const verb = (e.detail.requestConfig.verb || "").toLowerCase();
    const isMutation = verb === "post" || verb === "put" || verb === "delete";
    if (!isMutation) return;

    e.detail.requestConfig.headers = e.detail.requestConfig.headers || {};
    let cmdID = e.detail.requestConfig.headers["X-Command-Id"];
    if (!cmdID && typeof crypto !== "undefined" && crypto.randomUUID) {
      cmdID = crypto.randomUUID();
      e.detail.requestConfig.headers["X-Command-Id"] = cmdID;
    }
    if (!cmdID) return;

    const target = e.detail.elt;
    const syncEl =
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
  document.addEventListener("htmx:responseError", (e) => {
    const target = e.detail.elt;
    const syncEl = target.closest("[data-command-id]") || target.closest("[data-sync-state]");
    if (syncEl) {
      setSyncState(syncEl, "rejected");
      sync.pending = Math.max(0, sync.pending - 1);
      sync.failed++;
      announce(syncEl, "Network error — change not saved", true);
      updateIndicator();
    }
  });

  // --- Network error (offline): queue for retry instead of rejecting ---
  // htmx:sendError fires when the request can't be sent at all (network down).
  // Offline ≠ rejected — the command is queued, not lost.
  document.addEventListener("htmx:sendError", (e) => {
    const target = e.detail.elt;
    const syncEl = target.closest("[data-command-id]") || target.closest("[data-sync-state]");
    if (!syncEl) return;

    const cmdID = syncEl.getAttribute("data-command-id");
    if (!cmdID) return;

    // Capture the request envelope so the SharedWorker can persist it (ADR-0040)
    // and any tab can rebuild the request on cross-session retry.
    const cfg = (e.detail && e.detail.requestConfig) || null;
    let envelope = null;
    if (cfg) {
      // HTMX 2.x: cfg.parameters is a FormData object, not a plain object.
      // postMessage cannot clone FormData (structured clone algorithm does not
      // support it). Convert to a plain {key: value} object before sending to
      // the SharedWorker. Without this, offline commands are silently lost.
      let params = cfg.parameters;
      if (params instanceof FormData) {
        const plain = {};
        params.forEach((val, key) => { plain[key] = val; });
        params = plain;
      }
      envelope = {
        verb: (cfg.verb || "").toUpperCase(),
        url: cfg.path || "",
        values: params || null,
        headers: cfg.headers || null,
      };
    }

    // Mark as queued (offline) — NOT rejected
    syncEl.setAttribute("data-sync-queued", "");
    sync.pending = Math.max(0, sync.pending - 1);
    enqueueCommand(cmdID, envelope);
    announce(syncEl, "Offline — change queued for sync", true);
  });

  // --- Retry button: re-dispatch a rejected command ---
  document.addEventListener("click", (e) => {
    const btn = e.target.closest && e.target.closest("[data-sync-retry]");
    if (!btn) return;

    const row = btn.closest("[data-command-id]");
    if (!row) return;

    // Clear rejected state and re-trigger via HTMX if the original element exists
    setSyncState(row, "pending");
    sync.failed = Math.max(0, sync.failed - 1);
    sync.pending++;
    updateIndicator();

    // If the row has an hx-post/hx-get, re-issue it
    const trigger = row.querySelector("[hx-post], [hx-get]");
    if (trigger && typeof htmx !== "undefined") {
      htmx.trigger(trigger, "retry");
    }
  });

  // --- Boot: connect SSE + init offline queue on DOMContentLoaded ---
  function boot() {
    connectSSE();
    initSyncWorker();
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
