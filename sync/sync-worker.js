// sync-worker.js — SharedWorker for offline command queue (ADR 0029 + ADR 0040).
//
// One instance per origin, shared across all tabs. Queues command envelopes
// when the network is down, persists them to IndexedDB so they survive closed
// tabs / browser restarts, and tells tabs to retry when connectivity returns.
//
// The worker is a COORDINATOR, not a proxy:
//   - It does NOT send HTTP requests (tabs do, via HTMX).
//   - It does NOT own the SSE connection (tabs keep per-tab EventSource).
//   - It DOES persist queued commands to IndexedDB (ADR-0040 reverses the
//     ADR-0030 rejection: writes now survive closed tabs).
//
// Message protocol (tab <-> worker):
//   tab -> worker:  { type: "hello", tabId }
//                   Registers the tab. Must be the first message. If tabId
//                   already exists (page reload), the old port is replaced.
//   tab -> worker:  { type: "bye", tabId }
//                   Unregisters the tab (sent on beforeunload, best-effort).
//   tab -> worker:  { type: "enqueue", commandId, envelope }
//                   envelope = { verb, url, values, headers } — enough to
//                   rebuild the HTMX request. Commands missing a valid
//                   envelope are rejected as dead (cannot retry without one).
//   tab -> worker:  { type: "ack", commandId }
//                   Command confirmed or permanently rejected by the server.
//                   Deletes the persisted copy so it is not retried.
//   worker -> tab:  { type: "retry", commandId, envelope }
//                   Re-issue the command via HTMX. Delivered to the
//                   originating tab when possible, round-robin otherwise.
//   worker -> tab:  { type: "online" } | { type: "offline" }
//   worker -> tab:  { type: "pending", count }
//                   Persisted-queue depth for the UI sync indicator.
//   worker -> tab:  { type: "dead", commandId }
//                   Command exceeded MAX_RETRIES or RETRY_TTL_MS — the worker
//                   gave up. The tab should show it as permanently failed.
//
// Persistence model: IndexedDB DB "cqrshtmx-sync", object store "commands",
// keyPath "commandId". Each record: { commandId, envelope, queuedAt, retries }.
// IndexedDB is the single source of truth — there is no parallel in-memory
// queue. If IndexedDB is unavailable (private browsing, quota), the worker
// degrades to an in-memory Map that survives only the current worker lifetime.
//
// Eviction: commands exceeding MAX_RETRIES or RETRY_TTL_MS are marked dead
// (worker sends {type:"dead"} and deletes the record), preventing poison
// commands from retrying forever.
//
// Port management: tabs are tracked by tabId in a Map. On "hello", a tabId
// that already exists replaces the old port (handles page reload). On "bye",
// the entry is removed. Dead tabs (crash without bye) leave a stale entry
// whose postMessage is silently dropped — bounded by crash count, cleaned up
// when the worker is killed (all tabs closed).
//
// CONFIGURATION: The retry limits and stagger delays below are compile-time
// constants. To customize them, copy this file, modify the values, and serve
// the result via cqrshtmx.SyncWorkerHandlerWith(customJS, "1.0.0-custom").
// @ts-check
"use strict";

(function () {
  const VERSION = "1.2.0";

  // --- Configuration constants ---
  // To customize: copy this file, change values, serve via SyncWorkerHandlerWith.
  const DB_NAME = "cqrshtmx-sync";
  const STORE = "commands";
  const DB_VERSION = 1;
  const MAX_RETRIES = 10;
  const RETRY_TTL_MS = 24 * 60 * 60 * 1000; // 24 hours
  const STAGGER_MS = 100; // delay between successive retry messages
  const STAGGER_CAP_MS = 2000; // max stagger delay

  let db = null; // IDBDatabase once opened; null while opening or unavailable

  // tabId -> MessagePort. Tracks connected tabs for targeted retry and
  // broadcast. Dead tabs (crash without bye) leave a stale entry whose
  // postMessage is silently dropped — bounded by crash count, cleaned up
  // when the worker is killed (all tabs closed).
  const ports = new Map();

  // port -> tabId (WeakMap: entry is GC'd when the port is collected).
  const portTabId = new WeakMap();

  // commandId -> tabId. Tracks which tab enqueued each command so retry
  // messages go to the originating tab (avoids thundering herd: one tab
  // retries instead of every tab).
  const originatingTab = new Map();

  // In-memory fallback when IndexedDB is unavailable (private browsing,
  // quota exceeded). commandId -> { envelope, queuedAt, retries }.
  const memQueue = new Map();

  let online = navigator.onLine;
  let rrIndex = 0; // round-robin counter for distributing retried commands
  let flushing = false; // prevents concurrent flush cycles
  let flushPending = false; // schedules a follow-up flush after the current one

  // ---------------------------------------------------------------------------
  // Broadcasting
  // ---------------------------------------------------------------------------

  /**
   * Send a message to every connected tab. Tabs whose port throws on
   * postMessage are removed (crash detection).
   * @param {Record<string, unknown>} msg - Message to broadcast.
   */
  function broadcast(msg) {
    const dead = [];
    ports.forEach((port, tabId) => {
      try {
        port.postMessage(msg);
      } catch (e) {
        dead.push(tabId);
      }
    });
    for (const tabId of dead) {
      ports.delete(tabId);
    }
  }

  function alivePorts() {
    const list = [];
    ports.forEach((port) => {
      list.push(port);
    });
    return list;
  }

  // Picks the best port for a command: the originating tab if alive, else
  // round-robin across all alive ports.
  function pickPort(commandId, portList) {
    if (portList.length === 0) return null;
    const tabId = originatingTab.get(commandId);
    if (tabId && ports.has(tabId)) {
      return ports.get(tabId);
    }
    rrIndex = rrIndex % portList.length;
    return portList[rrIndex++];
  }

  // ---------------------------------------------------------------------------
  // IndexedDB helpers (all return Promises; failures degrade to in-memory)
  // ---------------------------------------------------------------------------

  function openDB() {
    return new Promise((resolve) => {
      if (typeof indexedDB === "undefined") {
        resolve(null);
        return;
      }
      try {
        const req = indexedDB.open(DB_NAME, DB_VERSION);
        req.onupgradeneeded = (e) => {
          const database = e.target.result;
          if (!database.objectStoreNames.contains(STORE)) {
            database.createObjectStore(STORE, { keyPath: "commandId" });
          }
        };
        req.onsuccess = (e) => {
          resolve(e.target.result);
        };
        req.onerror = () => {
          console.warn("[sync-worker] IndexedDB unavailable — degrading to in-memory");
          resolve(null);
        };
      } catch (e) {
        console.warn("[sync-worker] IndexedDB open failed:", e);
        resolve(null);
      }
    });
  }

  function idbRun(fn) {
    if (!db) return Promise.resolve(null);
    return new Promise((resolve) => {
      try {
        const tx = db.transaction(STORE, "readwrite");
        const store = tx.objectStore(STORE);
        const result = fn(store);
        tx.oncomplete = () => {
          resolve(result);
        };
        tx.onerror = () => {
          resolve(null);
        };
        tx.onabort = () => {
          resolve(null);
        };
      } catch (e) {
        console.warn("[sync-worker] IDB transaction failed:", e);
        resolve(null);
      }
    });
  }

  /**
   * Persist a command envelope to IndexedDB (or in-memory fallback).
   * Uses store.add (not put) so re-enqueue preserves retry count.
   * @param {string} commandId - Unique command identifier.
   * @param {{ verb: string, url: string, values: Object|null, headers: Object|null }} envelope - Request data for retry.
   * @returns {Promise<void>}
   */
  function persistCommand(commandId, envelope) {
    if (!db) {
      if (!memQueue.has(commandId)) {
        memQueue.set(commandId, {
          envelope: envelope,
          queuedAt: Date.now(),
          retries: 0,
        });
      }
      return Promise.resolve();
    }
    return idbRun((store) => {
      store.add({
        commandId: commandId,
        envelope: envelope,
        queuedAt: Date.now(),
        retries: 0,
      });
    });
  }

  /**
   * Delete a command from IndexedDB (or in-memory fallback).
   * Called on ACK (server confirmed/rejected) or eviction (dead).
   * @param {string} commandId - Unique command identifier.
   * @returns {Promise<void>}
   */
  function deleteCommand(commandId) {
    if (!db) {
      memQueue.delete(commandId);
      return Promise.resolve();
    }
    return idbRun((store) => {
      store.delete(commandId);
    });
  }

  function incrementRetryCount(commandId) {
    if (!db) {
      const memRecord = memQueue.get(commandId);
      if (memRecord) {
        memRecord.retries = (memRecord.retries || 0) + 1;
      }
      return Promise.resolve();
    }
    return new Promise((resolve) => {
      try {
        const tx = db.transaction(STORE, "readwrite");
        const store = tx.objectStore(STORE);
        const req = store.get(commandId);
        req.onsuccess = (e) => {
          const record = e.target.result;
          if (record) {
            record.retries = (record.retries || 0) + 1;
            store.put(record);
          }
        };
        tx.oncomplete = () => {
          resolve();
        };
        tx.onerror = () => {
          resolve();
        };
        tx.onabort = () => {
          resolve();
        };
      } catch (e) {
        resolve();
      }
    });
  }

  /**
   * Load all persisted commands from IndexedDB (or in-memory fallback).
   * @returns {Promise<Array<{ commandId: string, envelope: Object, queuedAt: number, retries: number }>>}
   */
  function loadAllCommands() {
    if (!db) {
      const result = [];
      memQueue.forEach((val, key) => {
        result.push({
          commandId: key,
          envelope: val.envelope,
          queuedAt: val.queuedAt,
          retries: val.retries,
        });
      });
      return Promise.resolve(result);
    }
    return new Promise((resolve) => {
      try {
        const tx = db.transaction(STORE, "readonly");
        const req = tx.objectStore(STORE).getAll();
        req.onsuccess = (e) => {
          resolve(e.target.result || []);
        };
        req.onerror = () => {
          resolve([]);
        };
      } catch (e) {
        resolve([]);
      }
    });
  }

  /**
   * Count pending commands in IndexedDB (or in-memory fallback).
   * Uses store.count() — not getAll().length — for efficiency.
   * @returns {Promise<number>}
   */
  function pendingCount() {
    if (!db) return Promise.resolve(memQueue.size);
    return new Promise((resolve) => {
      try {
        const tx = db.transaction(STORE, "readonly");
        const req = tx.objectStore(STORE).count();
        req.onsuccess = (e) => {
          resolve(e.target.result || 0);
        };
        req.onerror = () => {
          resolve(0);
        };
      } catch (e) {
        resolve(0);
      }
    });
  }

  function broadcastPendingCount() {
    pendingCount().then((count) => {
      broadcast({ type: "pending", count: count });
    });
  }

  // ---------------------------------------------------------------------------
  // Flush: read all persisted commands, evict dead ones, distribute retry
  // messages across alive tabs with staggered delivery.
  // ---------------------------------------------------------------------------

  /**
   * Trigger a flush cycle: read all persisted commands, evict dead ones
   * (MAX_RETRIES or TTL exceeded), then distribute retry messages to
   * alive tabs with staggered delivery (prevents thundering herd).
   * Concurrent calls are coalesced (flushPending flag).
   */
  function flush() {
    if (flushing) {
      flushPending = true;
      return;
    }
    flushing = true;
    doFlush().then(() => {
      flushing = false;
      if (flushPending) {
        flushPending = false;
        flush();
      }
    });
  }

  function doFlush() {
    return loadAllCommands().then((all) => {
      if (all.length === 0) {
        broadcastPendingCount();
        return;
      }

      const now = Date.now();
      const alive = [];

      for (const cmd of all) {
        const age = now - (cmd.queuedAt || now);
        const retries = cmd.retries || 0;

        if (age > RETRY_TTL_MS || retries >= MAX_RETRIES) {
          broadcast({ type: "dead", commandId: cmd.commandId });
          deleteCommand(cmd.commandId);
          originatingTab.delete(cmd.commandId);
        } else {
          alive.push(cmd);
        }
      }

      if (alive.length === 0) {
        broadcastPendingCount();
        return;
      }

      const portList = alivePorts();
      if (portList.length === 0) {
        // No tabs open — leave in IDB for next spawn
        broadcastPendingCount();
        return;
      }

      // Oldest first so long-waited commands get priority
      alive.sort((a, b) => (a.queuedAt || 0) - (b.queuedAt || 0));

      // Increment all retry counts before delivering so a concurrent
      // flush (after the lock releases) sees updated counts and does
      // not double-increment the same commands.
      const increments = [];
      for (let j = 0; j < alive.length; j++) {
        const item = alive[j];
        const port = pickPort(item.commandId, portList);
        if (!port) continue;

        increments.push(incrementRetryCount(item.commandId));

        // Stagger delivery to avoid thundering herd on server recovery
        const delay = Math.min(j * STAGGER_MS, STAGGER_CAP_MS);
        sendRetry(port, item, delay);
      }

      broadcastPendingCount();
      return Promise.all(increments);
    });
  }

  function sendRetry(port, cmd, delay) {
    setTimeout(() => {
      try {
        port.postMessage({
          type: "retry",
          commandId: cmd.commandId,
          envelope: cmd.envelope,
        });
      } catch (e) {
        // Port died — cleaned up on next broadcast
      }
    }, delay);
  }

  // ---------------------------------------------------------------------------
  // Boot: open IndexedDB, then flush if online so reconnecting tabs
  // immediately re-flight persisted commands.
  // ---------------------------------------------------------------------------

  openDB().then((database) => {
    db = database;
    if (online) {
      flush();
    } else {
      broadcastPendingCount();
    }
  });

  // ---------------------------------------------------------------------------
  // Connection handler: each tab connects via its own MessagePort
  // ---------------------------------------------------------------------------

  self.onconnect = (e) => {
    const port = e.ports[0];

    port.onmessage = (ev) => {
      const data = ev.data;
      if (!data || !data.type) return;

      if (data.type === "hello") {
        // Tab registered. If the same tabId reconnects (page reload),
        // replace the old port — prevents duplicate messages to the tab.
        if (ports.has(data.tabId)) {
          try {
            ports.get(data.tabId).close();
          } catch (err) {
            /* already dead */
          }
        }
        ports.set(data.tabId, port);
        portTabId.set(port, data.tabId);

        // Notify the tab of current online status immediately.
        port.postMessage({ type: online ? "online" : "offline" });

        // Always flush on connect. A newly loaded tab is online by definition,
        // and pending commands from a previous session should be retried immediately.
        // The online variable may be stale in some browser contexts (e.g. Playwright
        // SharedWorker scope doesn't always receive connectivity events).
        flush();
        return;
      }

      if (data.type === "bye") {
        const tabId = portTabId.get(port);
        if (tabId) ports.delete(tabId);
        portTabId.delete(port);
        return;
      }

      if (data.type === "enqueue") {
        if (!data.commandId) return;

        // Guard: don't persist commands without a valid envelope —
        // they can't be retried and would become poison entries.
        if (!data.envelope || !data.envelope.url || !data.envelope.verb) {
          port.postMessage({ type: "dead", commandId: data.commandId });
          return;
        }

        // Track which tab enqueued this for targeted retry
        originatingTab.set(data.commandId, portTabId.get(port));

        persistCommand(data.commandId, data.envelope).then(() => {
          broadcastPendingCount();
          // If already online, immediately attempt retry so commands
          // enqueued during a brief connectivity blip are re-flown.
          if (online) {
            flush();
          }
        });
        return;
      }

      if (data.type === "ack") {
        if (data.commandId) {
          originatingTab.delete(data.commandId);
          deleteCommand(data.commandId).then(broadcastPendingCount);
        }
        return;
      }
    };
  };

  // ---------------------------------------------------------------------------
  // Connectivity events: SharedWorker scope receives these directly
  // ---------------------------------------------------------------------------

  self.addEventListener("online", () => {
    online = true;
    broadcast({ type: "online" });
    flush();
  });

  self.addEventListener("offline", () => {
    online = false;
    broadcast({ type: "offline" });
  });
})();
