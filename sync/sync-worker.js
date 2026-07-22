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
"use strict";

(function () {
  var DB_NAME = "cqrshtmx-sync";
  var STORE = "commands";
  var DB_VERSION = 1;
  var MAX_RETRIES = 10;
  var RETRY_TTL_MS = 24 * 60 * 60 * 1000; // 24 hours
  var STAGGER_MS = 100; // delay between successive retry messages
  var STAGGER_CAP_MS = 2000; // max stagger delay

  var db = null; // IDBDatabase once opened; null while opening or unavailable

  // tabId -> MessagePort. Tracks connected tabs for targeted retry and
  // broadcast. Dead tabs (crash without bye) leave a stale entry whose
  // postMessage is silently dropped — bounded by crash count, cleaned up
  // when the worker is killed (all tabs closed).
  var ports = new Map();

  // port -> tabId (WeakMap: entry is GC'd when the port is collected).
  var portTabId = new WeakMap();

  // commandId -> tabId. Tracks which tab enqueued each command so retry
  // messages go to the originating tab (avoids thundering herd: one tab
  // retries instead of every tab).
  var originatingTab = new Map();

  // In-memory fallback when IndexedDB is unavailable (private browsing,
  // quota exceeded). commandId -> { envelope, queuedAt, retries }.
  var memQueue = new Map();

  var online = navigator.onLine;
  var rrIndex = 0; // round-robin counter for distributing retried commands
  var flushing = false; // prevents concurrent flush cycles
  var flushPending = false; // schedules a follow-up flush after the current one

  // ---------------------------------------------------------------------------
  // Broadcasting
  // ---------------------------------------------------------------------------

  function broadcast(msg) {
    var dead = [];
    ports.forEach(function (port, tabId) {
      try {
        port.postMessage(msg);
      } catch (e) {
        dead.push(tabId);
      }
    });
    for (var i = 0; i < dead.length; i++) {
      ports.delete(dead[i]);
    }
  }

  function alivePorts() {
    var list = [];
    ports.forEach(function (port) {
      list.push(port);
    });
    return list;
  }

  // Picks the best port for a command: the originating tab if alive, else
  // round-robin across all alive ports.
  function pickPort(commandId, portList) {
    if (portList.length === 0) return null;
    var tabId = originatingTab.get(commandId);
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
    return new Promise(function (resolve) {
      if (typeof indexedDB === "undefined") {
        resolve(null);
        return;
      }
      try {
        var req = indexedDB.open(DB_NAME, DB_VERSION);
        req.onupgradeneeded = function (e) {
          var database = e.target.result;
          if (!database.objectStoreNames.contains(STORE)) {
            database.createObjectStore(STORE, { keyPath: "commandId" });
          }
        };
        req.onsuccess = function (e) {
          resolve(e.target.result);
        };
        req.onerror = function () {
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
    return new Promise(function (resolve) {
      try {
        var tx = db.transaction(STORE, "readwrite");
        var store = tx.objectStore(STORE);
        var result = fn(store);
        tx.oncomplete = function () {
          resolve(result);
        };
        tx.onerror = function () {
          resolve(null);
        };
        tx.onabort = function () {
          resolve(null);
        };
      } catch (e) {
        console.warn("[sync-worker] IDB transaction failed:", e);
        resolve(null);
      }
    });
  }

  // Uses store.add (not store.put) so a re-enqueue of an existing command
  // preserves its retry count instead of resetting it to 0.
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
    return idbRun(function (store) {
      store.add({
        commandId: commandId,
        envelope: envelope,
        queuedAt: Date.now(),
        retries: 0,
      });
    });
  }

  function deleteCommand(commandId) {
    if (!db) {
      memQueue.delete(commandId);
      return Promise.resolve();
    }
    return idbRun(function (store) {
      store.delete(commandId);
    });
  }

  function incrementRetryCount(commandId) {
    if (!db) {
      var memRecord = memQueue.get(commandId);
      if (memRecord) {
        memRecord.retries = (memRecord.retries || 0) + 1;
      }
      return Promise.resolve();
    }
    return new Promise(function (resolve) {
      try {
        var tx = db.transaction(STORE, "readwrite");
        var store = tx.objectStore(STORE);
        var req = store.get(commandId);
        req.onsuccess = function (e) {
          var record = e.target.result;
          if (record) {
            record.retries = (record.retries || 0) + 1;
            store.put(record);
          }
        };
        tx.oncomplete = function () {
          resolve();
        };
        tx.onerror = function () {
          resolve();
        };
        tx.onabort = function () {
          resolve();
        };
      } catch (e) {
        resolve();
      }
    });
  }

  function loadAllCommands() {
    if (!db) {
      var result = [];
      memQueue.forEach(function (val, key) {
        result.push({
          commandId: key,
          envelope: val.envelope,
          queuedAt: val.queuedAt,
          retries: val.retries,
        });
      });
      return Promise.resolve(result);
    }
    return new Promise(function (resolve) {
      try {
        var tx = db.transaction(STORE, "readonly");
        var req = tx.objectStore(STORE).getAll();
        req.onsuccess = function (e) {
          resolve(e.target.result || []);
        };
        req.onerror = function () {
          resolve([]);
        };
      } catch (e) {
        resolve([]);
      }
    });
  }

  function pendingCount() {
    if (!db) return Promise.resolve(memQueue.size);
    return new Promise(function (resolve) {
      try {
        var tx = db.transaction(STORE, "readonly");
        var req = tx.objectStore(STORE).count();
        req.onsuccess = function (e) {
          resolve(e.target.result || 0);
        };
        req.onerror = function () {
          resolve(0);
        };
      } catch (e) {
        resolve(0);
      }
    });
  }

  function broadcastPendingCount() {
    pendingCount().then(function (count) {
      broadcast({ type: "pending", count: count });
    });
  }

  // ---------------------------------------------------------------------------
  // Flush: read all persisted commands, evict dead ones, distribute retry
  // messages across alive tabs with staggered delivery.
  // ---------------------------------------------------------------------------

  function flush() {
    if (flushing) {
      flushPending = true;
      return;
    }
    flushing = true;
    doFlush().then(function () {
      flushing = false;
      if (flushPending) {
        flushPending = false;
        flush();
      }
    });
  }

  function doFlush() {
    return loadAllCommands().then(function (all) {
      if (all.length === 0) {
        broadcastPendingCount();
        return;
      }

      var now = Date.now();
      var alive = [];

      for (var i = 0; i < all.length; i++) {
        var cmd = all[i];
        var age = now - (cmd.queuedAt || now);
        var retries = cmd.retries || 0;

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

      var portList = alivePorts();
      if (portList.length === 0) {
        // No tabs open — leave in IDB for next spawn
        broadcastPendingCount();
        return;
      }

      // Oldest first so long-waited commands get priority
      alive.sort(function (a, b) {
        return (a.queuedAt || 0) - (b.queuedAt || 0);
      });

      // Increment all retry counts before delivering so a concurrent
      // flush (after the lock releases) sees updated counts and does
      // not double-increment the same commands.
      var increments = [];
      for (var j = 0; j < alive.length; j++) {
        var item = alive[j];
        var port = pickPort(item.commandId, portList);
        if (!port) continue;

        increments.push(incrementRetryCount(item.commandId));

        // Stagger delivery to avoid thundering herd on server recovery
        var delay = Math.min(j * STAGGER_MS, STAGGER_CAP_MS);
        sendRetry(port, item, delay);
      }

      broadcastPendingCount();
      return Promise.all(increments);
    });
  }

  function sendRetry(port, cmd, delay) {
    setTimeout(function () {
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

  openDB().then(function (database) {
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

  self.onconnect = function (e) {
    var port = e.ports[0];

    port.onmessage = function (ev) {
      var data = ev.data;
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

        // If online, flush so a reconnecting tab immediately re-flights
        // persisted commands from a previous worker lifetime.
        if (online) {
          flush();
        } else {
          broadcastPendingCount();
        }
        return;
      }

      if (data.type === "bye") {
        var tabId = portTabId.get(port);
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

        persistCommand(data.commandId, data.envelope).then(function () {
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

  self.addEventListener("online", function () {
    online = true;
    broadcast({ type: "online" });
    flush();
  });

  self.addEventListener("offline", function () {
    online = false;
    broadcast({ type: "offline" });
  });
})();
