// sync-worker.js — SharedWorker for offline command queue (ADR 0029).
//
// One instance per origin, shared across all tabs. Queues command IDs when
// the network is down, and tells tabs to retry when connectivity returns.
//
// The worker is a COORDINATOR, not a proxy:
//   - It does NOT send HTTP requests (tabs do, via HTMX).
//   - It does NOT own the SSE connection (tabs keep per-tab EventSource).
//   - It does NOT persist to disk (in-memory only — Queue-Only contract).
//
// Message protocol (tab ↔ worker):
//   tab → worker:  { type: "enqueue", commandId: "<uuid>" }
//   worker → tab:  { type: "retry",   commandId: "<uuid>" }
//   worker → tab:  { type: "online" } | { type: "offline" }
//
// See ADR-0029 for the full architecture rationale.
"use strict";

(function () {
  // --- Queue: [{commandId, port}] — in-memory, dies with the worker ---
  var queue = [];
  var online = navigator.onLine;

  // --- Port management: track connected tabs for broadcast ---
  var ports = [];

  function broadcast(msg) {
    for (var i = 0; i < ports.length; i++) {
      try {
        ports[i].postMessage(msg);
      } catch {
        // Port closed — will be cleaned up on next flush
      }
    }
  }

  // --- Connection handler: each tab connects via its own MessagePort ---
  self.onconnect = function (e) {
    var port = e.ports[0];
    ports.push(port);

    port.onmessage = function (ev) {
      var data = ev.data;
      if (!data || !data.type) return;

      if (data.type === "enqueue") {
        if (!data.commandId) return;
        queue.push({ commandId: data.commandId, port: port });
      }
    };

    // Notify the tab of current online status immediately
    port.postMessage({ type: online ? "online" : "offline" });

    // Start a flush if we're already online (catches race: tab queued
    // before worker connected, then worker comes online)
    if (online) flush();
  };

  // --- Flush: tell each originating tab to retry its queued command ---
  function flush() {
    while (queue.length > 0) {
      var item = queue.shift();
      try {
        item.port.postMessage({ type: "retry", commandId: item.commandId });
      } catch {
        // Port closed — command is lost (acceptable: Queue-Only)
      }
    }
  }

  // --- Connectivity events: SharedWorker scope receives these directly ---
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
