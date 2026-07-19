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
//   tab -> worker:  { type: "enqueue", commandId, envelope }
//                   envelope = { verb, url, values, headers } (enough to rebuild the HTMX request)
//   tab -> worker:  { type: "ack", commandId }      // command confirmed by server -> delete from IDB
//   worker -> tab:  { type: "retry", commandId, envelope }
//   worker -> tab:  { type: "online" } | { type: "offline" }
//   worker -> tab:  { type: "pending", count }      // persisted-queue depth for UI indicator
//
// Persistence model: IndexedDB DB "cqrshtmx-sync", object store "commands",
// keyPath "commandId". On spawn the worker drains all pending commands and
// broadcasts retry to every connected tab. On ACK the command is deleted. If
// IndexedDB is unavailable (private browsing, quota), the worker degrades
// gracefully to in-memory-only behavior.
"use strict";

(function () {
  var DB_NAME = "cqrshtmx-sync";
  var STORE = "commands";
  var DB_VERSION = 1;
  var db = null; // IDBDatabase once opened; null while opening or unavailable

  // --- In-memory queue: [{commandId, envelope, port}] for the current worker lifetime ---
  var queue = [];
  var online = navigator.onLine;

  // --- Port management: track connected tabs for broadcast ---
  var ports = [];

  function broadcast(msg) {
    for (var i = 0; i < ports.length; i++) {
      try {
        ports[i].postMessage(msg);
      } catch (e) {
        // Port closed — cleaned up on next flush
      }
    }
  }

  // --- IndexedDB helpers (all return Promises; failures degrade to in-memory) ---

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
          resolve(null); // private mode / blocked — degrade gracefully
        };
      } catch (e) {
        resolve(null);
      }
    });
  }

  function idbRun(fn) {
    if (!db) return Promise.resolve();
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
        resolve(null);
      }
    });
  }

  function persistCommand(entry) {
    return idbRun(function (store) {
      // Structured-clone-safe record (ports are NOT cloneable, so omit them).
      store.put({
        commandId: entry.commandId,
        envelope: entry.envelope,
        queuedAt: Date.now(),
      });
    });
  }

  function deleteCommand(commandId) {
    return idbRun(function (store) {
      store.delete(commandId);
    });
  }

  function loadAllCommands() {
    return new Promise(function (resolve) {
      if (!db) {
        resolve([]);
        return;
      }
      try {
        var tx = db.transaction(STORE, "readonly");
        var store = tx.objectStore(STORE);
        var req = store.getAll();
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

  function broadcastPendingCount() {
    loadAllCommands().then(function (all) {
      broadcast({ type: "pending", count: all.length });
    });
  }

  // --- Flush: tell each originating tab to retry its queued command ---
  function flush() {
    while (queue.length > 0) {
      var item = queue.shift();
      try {
        item.port.postMessage({
          type: "retry",
          commandId: item.commandId,
          envelope: item.envelope,
        });
      } catch (e) {
        // Port closed — the persisted copy remains in IDB and will be
        // re-drained on the next spawn (cross-tab retry).
      }
    }
  }

  // --- Spawn drain: on first connect, replay anything persisted from a
  //     previous worker lifetime (closed tabs, browser restart). These have no
  //     originating port, so broadcast retry to every connected tab. ---
  function drainPersisted() {
    loadAllCommands().then(function (all) {
      if (all.length === 0) {
        broadcastPendingCount();
        return;
      }
      for (var i = 0; i < all.length; i++) {
        broadcast({
          type: "retry",
          commandId: all[i].commandId,
          envelope: all[i].envelope,
        });
      }
      broadcastPendingCount();
    });
  }

  // --- Boot: open IndexedDB, then drain if online so reconnecting tabs
  //     immediately re-flight persisted commands. ---
  openDB().then(function (database) {
    db = database;
    if (online) {
      drainPersisted();
    } else {
      broadcastPendingCount();
    }
  });

  // --- Connection handler: each tab connects via its own MessagePort ---
  self.onconnect = function (e) {
    var port = e.ports[0];
    ports.push(port);

    port.onmessage = function (ev) {
      var data = ev.data;
      if (!data || !data.type) return;

      if (data.type === "enqueue") {
        if (!data.commandId) return;
        var entry = {
          commandId: data.commandId,
          envelope: data.envelope || null,
          port: port,
        };
        queue.push(entry);
        persistCommand(entry).then(broadcastPendingCount);
      } else if (data.type === "ack") {
        // Command confirmed (or permanently rejected) by the server — remove
        // from the durable queue so it is not retried again.
        if (data.commandId) {
          deleteCommand(data.commandId).then(broadcastPendingCount);
        }
      }
    };

    // Notify the tab of current online status immediately.
    port.postMessage({ type: online ? "online" : "offline" });

    // Start a flush if we're already online (catches the race: tab queued
    // before the worker connected, then the worker comes online).
    if (online) {
      drainPersisted();
      flush();
    }
  };

  // --- Connectivity events: SharedWorker scope receives these directly ---
  self.addEventListener("online", function () {
    online = true;
    broadcast({ type: "online" });
    drainPersisted();
    flush();
  });

  self.addEventListener("offline", function () {
    online = false;
    broadcast({ type: "offline" });
  });
})();
