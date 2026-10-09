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
  const VERSION = "1.5.0";

  // --- Sync protocol endpoints (ADR-0056) ---
  // Pull (reads): cursor-based event batches cached in the worker. Push
  // (writes): batched queue flush for commands stamped with a command type.
  // Configure per-app via <body data-sync-pull-url data-sync-push-url>.
  const DEFAULT_PULL_URL = "/sync/pull";
  const DEFAULT_PUSH_URL = "/sync/push";
  const PULL_PAGE_LIMIT = 500;
  const PULL_MAX_PAGES = 20;

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

  // --- Persistent client ID (offline-first attribution) ---
  // One ULID per browser, persisted in localStorage, stamped as X-Client-Id
  // on every mutation so server-side events can attribute offline-originated
  // commands to the device that created them (go-cqrs-lite metadata key
  // "client.id"). ULID format: 26-char Crockford Base32 (timestamp + random),
  // parseable by the server's id.ParseClientID — a crypto.randomUUID UUID
  // would be rejected.
  // Must match the Go constants cqrshtmx.HeaderClientID ("X-Client-Id") and
  // cqrshtmx.CommandIDHeader ("X-Command-Id") — keep them in sync manually
  // (no codegen link across languages).
  const CLIENT_ID_HEADER = "X-Client-Id";
  const COMMAND_ID_HEADER = "X-Command-Id";
  const CLIENT_ID_KEY = "cqrs-htmx-client-id";

  /**
   * Generate a ULID (26 chars, Crockford Base32, ULID-spec compatible).
   * 48-bit millisecond timestamp + 80 random bits from crypto.getRandomValues.
   * @returns {string} A new ULID string.
   */
  function ulid() {
    const ENC = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";
    let time = Date.now();
    let timePart = "";
    for (let i = 0; i < 10; i++) {
      timePart = ENC[time % 32] + timePart;
      time = Math.floor(time / 32);
    }
    const bytes = new Uint8Array(16);
    if (typeof crypto !== "undefined" && crypto.getRandomValues) {
      crypto.getRandomValues(bytes);
    } else {
      for (let i = 0; i < 16; i++) bytes[i] = (Math.random() * 256) | 0;
    }
    let randPart = "";
    for (let i = 0; i < 16; i++) randPart += ENC[bytes[i] % 32];
    return timePart + randPart;
  }

  /**
   * Get (or lazily create) the persistent client ID for this browser.
   * Survives restarts via localStorage; best-effort if storage is unavailable
   * (private mode quotas) — falls back to a per-load ID.
   * @returns {string} A ULID identifying this client device.
   */
  function getClientID() {
    try {
      let cid = localStorage.getItem(CLIENT_ID_KEY);
      if (!cid || !/^[0-9A-HJKMNP-TV-Z]{26}$/.test(cid)) {
        cid = ulid();
        localStorage.setItem(CLIENT_ID_KEY, cid);
      }
      return cid;
    } catch (e) {
      // localStorage unavailable — per-session ID is still useful attribution
      return ulid();
    }
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
  let sseHadError = false;
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
      // After a dropped connection the client may have missed events —
      // catch up from the cached cursor (ADR-0056 pull).
      if (sseHadError) {
        sseHadError = false;
        pullEvents();
      }
    });

    eventSource.onerror = () => {
      sseHadError = true;
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
        } else if (data.type === "retry-batch") {
          pushCommandBatch(data.commands || []);
        } else if (data.type === "pending") {
          sync.queued = Math.max(sync.queued, data.count | 0);
          updateIndicator();
        } else if (data.type === "dead") {
          handleDeadCommand(data.commandId);
        } else if (data.type === "sync:events") {
          notifyEventsCached(data.cursor || "", data.count | 0, false);
        } else if (data.type === "sync:reset") {
          notifyEventsCached("", 0, true);
        } else if (data.type === "sync-state") {
          if (pendingStateResolve) {
            pendingStateResolve({ cursor: data.cursor || "", backendId: data.backendId || "" });
            pendingStateResolve = null;
          }
        } else if (data.type === "events") {
          if (pendingEventsResolve) {
            pendingEventsResolve({ events: data.events || [], cursor: data.cursor || "", backendId: data.backendId || "" });
            pendingEventsResolve = null;
          }
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

      // When the page detects connectivity return: PULL FIRST (ADR-0056 —
      // catch up on missed events so queued commands re-decide against fresh
      // state), THEN tell the worker to flush the offline queue.
      window.addEventListener("online", () => {
        pullEvents().then(() => {
          if (syncWorker) {
            try {
              syncWorker.port.postMessage({ type: "flush" });
            } catch (e) {
              // Worker gone — nothing to flush
            }
          }
        });
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

  // --- ADR-0056 read half: cursor-based event pull + worker cache ---

  let pendingStateResolve = null;
  let pendingEventsResolve = null;

  /**
   * Ask the worker for the persisted sync state (cursor + backendId).
   * @returns {Promise<{ cursor: string, backendId: string }>}
   */
  function getSyncState() {
    if (!syncWorker) return Promise.resolve({ cursor: "", backendId: "" });
    return new Promise((resolve) => {
      pendingStateResolve = resolve;
      try {
        syncWorker.port.postMessage({ type: "get-state" });
        setTimeout(() => {
          if (pendingStateResolve === resolve) {
            pendingStateResolve = null;
            resolve({ cursor: "", backendId: "" });
          }
        }, 2000);
      } catch (e) {
        pendingStateResolve = null;
        resolve({ cursor: "", backendId: "" });
      }
    });
  }

  /**
   * Ask the worker for the cached event feed (offline reads).
   * @returns {Promise<{ events: Array<Object>, cursor: string, backendId: string }>}
   */
  function getCachedEvents() {
    if (!syncWorker) return Promise.resolve({ events: [], cursor: "", backendId: "" });
    return new Promise((resolve) => {
      pendingEventsResolve = resolve;
      try {
        syncWorker.port.postMessage({ type: "get-events" });
        setTimeout(() => {
          if (pendingEventsResolve === resolve) {
            pendingEventsResolve = null;
            resolve({ events: [], cursor: "", backendId: "" });
          }
        }, 2000);
      } catch (e) {
        pendingEventsResolve = null;
        resolve({ events: [], cursor: "", backendId: "" });
      }
    });
  }

  /**
   * Pull events from the server starting at the cached cursor, page until
   * caught up (or the page cap), hand the batch to the worker for caching,
   * and notify the document. Safe to call repeatedly; a pull in flight is
   * coalesced. Resolves when caught up (or on failure — never throws).
   * @returns {Promise<void>}
   */
  let pullInFlight = false;
  let pullQueued = false;
  function pullEvents() {
    if (!syncWorker || typeof fetch === "undefined") return Promise.resolve();
    if (pullInFlight) {
      pullQueued = true;
      return Promise.resolve();
    }
    pullInFlight = true;

    const pullURL = document.body.getAttribute("data-sync-pull-url") || DEFAULT_PULL_URL;

    return getSyncState()
      .then((state) => {
        const collected = [];
        let cursor = state.cursor;
        let backendId = state.backendId;

        function page() {
          const url = pullURL + "?limit=" + PULL_PAGE_LIMIT + (cursor ? "&after=" + encodeURIComponent(cursor) : "");
          return fetch(url, { credentials: "same-origin" }).then((res) => {
            if (!res.ok) throw new Error("pull failed: " + res.status);
            return res.json();
          }).then((body) => {
            backendId = body.backendId || backendId;
            for (let i = 0; i < (body.events || []).length; i++) {
              collected.push(body.events[i]);
            }
            cursor = body.nextCursor || cursor;
            if (body.hasMore && pageCount < PULL_MAX_PAGES) {
              pageCount++;
              return page();
            }
          });
        }

        let pageCount = 0;
        return page().then(() => {
          if (collected.length > 0 || cursor !== state.cursor) {
            syncWorker.port.postMessage({
              type: "cache-events",
              cursor: cursor,
              backendId: backendId,
              events: collected,
            });
          }
        });
      })
      .catch(() => {
        // Offline / server gone — the cached feed is still readable; the
        // next online/SSE-reconnect/batch-push completion re-triggers a pull.
      })
      .then(() => {
        pullInFlight = false;
        if (pullQueued) {
          pullQueued = false;
          pullEvents();
        }
      });
  }

  /**
   * Broadcast cached-events notifications to the document so consumer UIs
   * can re-render from the cache (htmx event + DOM CustomEvent).
   */
  function notifyEventsCached(cursor, count, reset) {
    const detail = { cursor: cursor, count: count, reset: !!reset };
    document.dispatchEvent(new CustomEvent("cqrshtmx:sync-events", { detail: detail }));
    if (typeof htmx !== "undefined") {
      htmx.trigger(document.body, "cqrshtmx:sync-events", detail);
    }
  }

  // --- ADR-0056 write half: batched command push ---

  /**
   * Serialize captured values into a request body string for a push envelope.
   * Form posts (the HTMX default) are URL-encoded; JSON endpoints get JSON.
   */
  function envelopeBody(envelope) {
    const values = envelope.values || {};
    if ((envelope.contentType || "").indexOf("json") !== -1) {
      return JSON.stringify(values);
    }
    return new URLSearchParams(values).toString();
  }

  /**
   * POST one batch of queued typed commands to /sync/push and reconcile the
   * per-command outcomes: confirmed/permanently-rejected commands are ACKed
   * (deleted from the queue); transient failures stay queued for the next
   * flush cycle. Afterwards, pull to pick up the resulting events.
   * @param {Array<{ commandId: string, envelope: Object }>} commands
   */
  function pushCommandBatch(commands) {
    if (commands.length === 0 || typeof fetch === "undefined") return;

    const pushURL = document.body.getAttribute("data-sync-push-url") || DEFAULT_PUSH_URL;

    // Headers from the first envelope carry session-level concerns
    // (X-Client-Id, CSRF token if the app put one in hx-headers).
    const headers = { "Content-Type": "application/json" };
    const firstHeaders = (commands[0].envelope && commands[0].envelope.headers) || {};
    for (const key of Object.keys(firstHeaders)) {
      const lower = key.toLowerCase();
      if (lower !== "content-type" && lower !== "content-length") {
        headers[key] = firstHeaders[key];
      }
    }

    const batch = commands.map((cmd) => ({
      commandId: cmd.commandId,
      type: cmd.envelope.commandType,
      body: envelopeBody(cmd.envelope),
      contentType: cmd.envelope.contentType || "application/x-www-form-urlencoded",
    }));

    fetch(pushURL, {
      method: "POST",
      credentials: "same-origin",
      headers: headers,
      body: JSON.stringify({ commands: batch }),
    })
      .then((res) => {
        if (!res.ok) throw new Error("push failed: " + res.status);
        return res.json();
      })
      .then((body) => {
        const results = body.results || [];
        for (let i = 0; i < results.length; i++) {
          const result = results[i];
          const family = result.error ? result.error.family : "";
          if (result.status === "confirmed" || (family !== "transient" && result.status === "rejected")) {
            handleSyncAck({
              commandId: result.commandId,
              status: result.status,
              error: result.error ? result.error.message : undefined,
            });
            ackCommand(result.commandId);
          }
          // transient rejections: stay queued — the worker's next flush
          // cycle retries them (retry count already incremented).
        }
        updateIndicator();
        // Pull-before-push's mirror: after pushing, catch up on the events
        // the just-confirmed commands produced.
        pullEvents();
      })
      .catch(() => {
        // Network failed mid-batch — every command stays queued; the next
        // flush cycle re-sends the batch.
      });
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
    let cmdID = e.detail.requestConfig.headers[COMMAND_ID_HEADER];
    if (!cmdID && typeof crypto !== "undefined" && crypto.randomUUID) {
      cmdID = crypto.randomUUID();
      e.detail.requestConfig.headers[COMMAND_ID_HEADER] = cmdID;
    }
    // Stamp the persistent client ID for offline-first attribution. Persists
    // into queued envelopes (headers are captured) so retries carry it too.
    e.detail.requestConfig.headers[CLIENT_ID_HEADER] = getClientID();
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
        params.forEach((val, key) => {
          plain[key] = val;
        });
        params = plain;
      }
      envelope = {
        verb: (cfg.verb || "").toUpperCase(),
        url: cfg.path || "",
        values: params || null,
        headers: cfg.headers || null,
        // ADR-0056 batch push: the issuing element (or an ancestor) can stamp
        // data-sync-command-type so the queue flush batches this command via
        // POST /sync/push instead of per-URL HTMX replay. Absent = classic path.
        commandType: (target.closest && target.closest("[data-sync-command-type]"))
          ? target.closest("[data-sync-command-type]").getAttribute("data-sync-command-type")
          : "",
        contentType: (cfg.headers && cfg.headers["Content-Type"]) || "application/x-www-form-urlencoded",
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

  // --- Boot: connect SSE + init offline queue + initial catch-up pull ---
  function boot() {
    connectSSE();
    initSyncWorker();
    // Initial pull: a freshly loaded tab catches up on everything missed
    // while it was closed (ADR-0056). The worker broadcast tells other tabs.
    pullEvents();
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }

  // --- Public API (ADR-0056): offline reads + manual pull for consumer UIs ---
  // window.cqrsSync.getEvents() — the cached event feed (survives offline).
  // window.cqrsSync.pull()     — force a catch-up pull; resolves when done.
  // window.cqrsSync.version    — client version (matches SyncVersion()).
  window.cqrsSync = {
    version: VERSION,
    getEvents: getCachedEvents,
    pull: pullEvents,
  };
})();
