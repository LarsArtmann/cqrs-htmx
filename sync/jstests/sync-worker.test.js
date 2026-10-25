// Protocol-level node:test suite for the embedded sync-worker.js (ADR-0056).
//
// The worker is a SharedWorker IIFE driven entirely by its message protocol,
// so these tests load the real shipped file into a sandbox with a minimal
// deterministic IndexedDB shim and talk to it over fake MessagePorts — the
// same seam a browser uses. Both v1.5.0 defects (keyless meta put dropping
// cursor persistence; backend-reset wipe semantics) are pinned here in
// milliseconds instead of Playwright battery runs. v1.6.0 adds the v1→v2
// schema-upgrade boot pins (in-place upgrade, pending-command survival, fresh
// full-schema boot) and the cursor-monotonicity guard pin (regression refused
// on the same backend, rewind allowed across a backendId reset).
//
// Run via the Go wrapper (js_worker_test.go) or directly:
//   node --test ./jstests/sync-worker.test.js   (from sync/)
"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

// ---------------------------------------------------------------------------
// Minimal IndexedDB shim: deterministic, synchronous mutations with
// async-fired (microtask) events, matching real IDB event ordering
// (request onsuccess before transaction oncomplete). Only the surface
// sync-worker.js touches: open with upgrade, transaction(stores, mode),
// get/put/add/delete/getAll/clear/count, and the DataError law for keyless
// puts on out-of-line-key stores.
// ---------------------------------------------------------------------------

class FakeRequest {
  constructor() {
    this.onsuccess = null;
    this.onerror = null;
    this.result = undefined;
    this.error = null;
  }
}

class FakeObjectStore {
  constructor(name, keyPath, data) {
    this.name = name;
    this.keyPath = keyPath; // null = out-of-line keys (caller must pass keys)
    this.data = data;
  }

  _keyFor(value, explicitKey) {
    if (explicitKey !== undefined) return explicitKey;
    if (this.keyPath) return value[this.keyPath];
    // Real IndexedDB throws DataError synchronously here.
    const err = new TypeError("DataError: no key supplied for out-of-line-key store");
    err.name = "DataError";
    throw err;
  }

  put(value, explicitKey) {
    const req = new FakeRequest();
    const key = this._keyFor(value, explicitKey);
    this.data.set(key, value);
    queueMicrotask(() => req.onsuccess && req.onsuccess({ target: req }));
    return req;
  }

  add(value) {
    return this.put(value);
  }

  get(key) {
    const req = new FakeRequest();
    req.result = this.data.has(key) ? this.data.get(key) : undefined;
    queueMicrotask(() => req.onsuccess && req.onsuccess({ target: req }));
    return req;
  }

  delete(key) {
    const req = new FakeRequest();
    this.data.delete(key);
    queueMicrotask(() => req.onsuccess && req.onsuccess({ target: req }));
    return req;
  }

  getAll() {
    const req = new FakeRequest();
    req.result = Array.from(this.data.values());
    queueMicrotask(() => req.onsuccess && req.onsuccess({ target: req }));
    return req;
  }

  clear() {
    const req = new FakeRequest();
    this.data.clear();
    queueMicrotask(() => req.onsuccess && req.onsuccess({ target: req }));
    return req;
  }

  count() {
    const req = new FakeRequest();
    req.result = this.data.size;
    queueMicrotask(() => req.onsuccess && req.onsuccess({ target: req }));
    return req;
  }
}

class FakeTransaction {
  constructor(stores) {
    this.stores = stores;
    this.oncomplete = null;
    this.onerror = null;
    this.onabort = null;
    // Transaction completion fires after every request scheduled inside it
    // (requests queue one microtask each; complete queues after them).
    let depth = 0;
    const fire = () => {
      if (depth++ < 2) {
        queueMicrotask(fire);
        return;
      }
      this.oncomplete && this.oncomplete();
    };
    queueMicrotask(fire);
  }

  objectStore(name) {
    return this.stores[name];
  }
}

class FakeDB {
  constructor(name) {
    this.name = name;
    this.version = 0;
    this.stores = new Map();
    this.objectStoreNames = {
      contains: (name) => this.stores.has(name),
    };
  }

  createObjectStore(name, opts) {
    this.stores.set(name, new FakeObjectStore(name, (opts && opts.keyPath) || null, new Map()));
    return this.stores.get(name);
  }

  transaction(storeNames, _mode) {
    const names = Array.isArray(storeNames) ? storeNames : [storeNames];
    const stores = {};
    for (const n of names) {
      if (!this.stores.has(n)) {
        throw new TypeError("NotFoundError: no such object store: " + n);
      }
      stores[n] = this.stores.get(n);
    }
    return new FakeTransaction(stores);
  }
}

class FakeIndexedDB {
  constructor() {
    this.databases = new Map();
  }

  open(name, version) {
    const req = new FakeRequest();
    queueMicrotask(() => {
      let db = this.databases.get(name);
      const fresh = !db;
      if (fresh) {
        db = new FakeDB(name);
        this.databases.set(name, db);
      }
      if (version !== undefined && version > db.version) {
        db.version = version;
        // Real IDB: the event target is the open request; its .result is the
        // database (the upgrade handler reads e.target.result.createObjectStore).
        req.onupgradeneeded && req.onupgradeneeded({ target: { result: db } });
      }
      req.result = db;
      req.onsuccess && req.onsuccess({ target: req });
    });
    return req;
  }
}

// ---------------------------------------------------------------------------
// Worker loader: sandbox with the shim + a self object, then drive the
// protocol through fake ports.
// ---------------------------------------------------------------------------

function loadWorker(options) {
  const opts = options && options.idb ? options : {};
  const workerSrc = fs.readFileSync(path.join(__dirname, "..", "sync-worker.js"), "utf8");
  const idb = opts.idb || new FakeIndexedDB();
  const warns = [];

  const sandbox = {
    indexedDB: idb,
    navigator: { onLine: true },
    console: {
      warn: function () {
        warns.push(Array.prototype.join.call(arguments, " "));
      },
      log: function () {},
    },
    setTimeout,
    clearTimeout,
    Date,
    Promise,
    queueMicrotask,
    self: null,
  };
  sandbox.self = {
    onconnect: null,
    listeners: {},
    addEventListener(type, fn) {
      (this.listeners[type] = this.listeners[type] || []).push(fn);
    },
  };

  const context = vm.createContext(sandbox);
  vm.runInContext(workerSrc, context, { filename: "sync-worker.js" });

  assert.ok(sandbox.self.onconnect, "worker did not register self.onconnect");

  return {
    idb,
    warns,
    ports: [],

    connect() {
      const received = [];
      const port = {
        onmessage: null,
        messages: received,
        closed: false,
        postMessage(msg) {
          received.push(msg);
        },
        close() {
          this.closed = true;
        },
      };
      sandbox.self.onconnect({ ports: [port] });
      this.ports.push(port);
      return port;
    },

    emit(type) {
      for (const fn of sandbox.self.listeners[type] || []) fn();
    },
  };
}

async function drain(ms = 25) {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

function waitFor(messages, predicate, label, timeoutMs = 500) {
  return new Promise((resolve, reject) => {
    const started = Date.now();
    const tick = () => {
      const found = messages.find(predicate);
      if (found) return resolve(found);
      if (Date.now() - started > timeoutMs) {
        return reject(
          new Error("timeout waiting for " + label + "; got: " + JSON.stringify(messages)),
        );
      }
      setTimeout(tick, 10);
    };
    tick();
  });
}

// A valid command envelope (the enqueue guard demands url + verb).
function envelope(overrides) {
  return Object.assign(
    {
      verb: "POST",
      url: "/api/notes",
      values: { name: "x" },
      headers: null,
      commandType: "CreateNote",
    },
    overrides,
  );
}

// askWorker connects a fresh port (the worker's onconnect installs its own
// onmessage handler on it — the same handler a browser MessagePort would
// dispatch through), sends one message, and awaits one typed reply.
async function askWorker(worker, message, wantedType) {
  const ask = worker.connect();
  ask.onmessage({ data: message });

  return waitFor(ask.messages, (m) => m.type === wantedType, wantedType + " reply");
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test("hello registers the tab, answers connectivity, and reports an empty queue", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  const kinds = port.messages.map((m) => m.type);
  assert.ok(kinds.includes("online") || kinds.includes("offline"), "connectivity answer missing");
  assert.ok(kinds.includes("pending"), "pending count missing");
  const pending = port.messages.find((m) => m.type === "pending");
  assert.equal(pending.count, 0);
});

test("enqueue persists a typed command and it flushes as ONE retry-batch", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  port.onmessage({
    data: {
      type: "enqueue",
      commandId: "cmd-1",
      envelope: envelope({ url: "/api/notes", values: { name: "a" } }),
    },
  });
  port.onmessage({
    data: { type: "enqueue", commandId: "cmd-2", envelope: envelope({ values: { name: "b" } }) },
  });
  await drain();

  const pending = port.messages.find((m) => m.type === "pending" && m.count === 2);
  assert.ok(pending, "pending count 2 missing: " + JSON.stringify(port.messages));

  port.onmessage({ data: { type: "flush" } });
  const batch = await waitFor(
    port.messages,
    (m) => m.type === "retry-batch",
    "retry-batch after flush",
  );
  assert.equal(batch.commands.length, 2, "both typed commands must ride ONE batch");
  assert.equal(
    batch.commands
      .map((c) => c.commandId)
      .sort()
      .join(","),
    "cmd-1,cmd-2",
  );

  // ack removes exactly the acknowledged command
  port.onmessage({ data: { type: "ack", commandId: "cmd-1" } });
  await drain();
  const after = port.messages.filter((m) => m.type === "pending").pop();
  assert.equal(after.count, 1, "ack must remove exactly its command");
});

test("enqueue without a valid envelope is refused (poison guard)", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  port.onmessage({ data: { type: "enqueue", commandId: "cmd-x", envelope: { url: "/no-verb" } } });
  const dead = await waitFor(port.messages, (m) => m.type === "dead", "dead verdict");
  assert.equal(dead.commandId, "cmd-x");
});

test("cache-events persists cursor + backendId (the v1.5.0 keyless-put defect pin)", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  port.onmessage({
    data: {
      type: "cache-events",
      cursor: "cursor-1",
      backendId: "backend-A",
      events: [{ eventId: "e1", type: "Note.created" }],
    },
  });
  await waitFor(port.messages, (m) => m.type === "sync:events", "sync:events broadcast");

  // THE regression: with the v1.5.0 keyless metaStore.put bug, the DataError
  // aborted the transaction and the state NEVER persisted — get-state came
  // back empty. It must now round-trip.
  const state = await askWorker(worker, { type: "get-state" }, "sync-state");
  assert.equal(state.cursor, "cursor-1", "cursor must survive the write (keyless-put regression)");
  assert.equal(state.backendId, "backend-A", "backendId must survive the write");
});

test("cached events read back offline and survive reconnect (persistence)", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  port.onmessage({
    data: {
      type: "cache-events",
      cursor: "c9",
      backendId: "b1",
      events: [
        { eventId: "e2", type: "Note.created" },
        { eventId: "e1", type: "Note.created" },
      ],
    },
  });
  await waitFor(port.messages, (m) => m.type === "sync:events", "sync:events broadcast");

  const read = () => askWorker(worker, { type: "get-events" }, "events");

  const first = await read();
  assert.equal(
    first.events.map((e) => e.eventId).join(","),
    "e1,e2", // sorted most-recent-last
  );

  const second = await read(); // a NEW connection reads the same cache
  assert.equal(
    second.events.map((e) => e.eventId).join(","),
    "e1,e2",
    "cache must persist across connections",
  );
});

test("backendId change wipes the cache and broadcasts sync:reset, keeping the queue", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  port.onmessage({ data: { type: "enqueue", commandId: "keep-me", envelope: envelope() } });
  await drain();

  port.onmessage({
    data: {
      type: "cache-events",
      cursor: "c1",
      backendId: "backend-A",
      events: [{ eventId: "e1" }],
    },
  });
  await waitFor(port.messages, (m) => m.type === "sync:events", "first cache write");

  port.onmessage({
    data: { type: "cache-events", cursor: "", backendId: "backend-B", events: [] },
  });
  await waitFor(port.messages, (m) => m.type === "sync:reset", "sync:reset on backend change");

  const read = await askWorker(worker, { type: "get-events" }, "events");
  assert.equal(read.events.length, 0, "cache must be wiped after a backend reset");

  const pending = port.messages.filter((m) => m.type === "pending").pop();
  assert.equal(pending.count, 1, "the command queue must SURVIVE a backend reset");
});

// ---------------------------------------------------------------------------
// v1 → v2 schema upgrade (v1.6.0): a 1.4.x database (commands store only,
// ADR-0040) must upgrade in place when the v2 worker opens it — events + meta
// stores created via onupgradeneeded, the pending v1 command preserved.
// ---------------------------------------------------------------------------

function seedV1Database(idb) {
  const legacy = new FakeDB("cqrshtmx-sync");
  legacy.version = 1;
  legacy.createObjectStore("commands", { keyPath: "commandId" });
  legacy.stores.get("commands").data.set("legacy-cmd", {
    commandId: "legacy-cmd",
    envelope: envelope(),
    queuedAt: Date.now(),
    retries: 0,
  });
  idb.databases.set("cqrshtmx-sync", legacy);
}

test("boot upgrades a v1 database to v2 in place, preserving the pending command", async () => {
  const idb = new FakeIndexedDB();
  seedV1Database(idb);
  const worker = loadWorker({ idb });
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  const db = worker.idb.databases.get("cqrshtmx-sync");
  assert.equal(db.version, 2, "the v1 database must be upgraded to DB_VERSION 2");
  assert.ok(db.objectStoreNames.contains("commands"), "the v1 commands store must survive");
  assert.ok(db.objectStoreNames.contains("events"), "the upgrade must create the events store");
  assert.ok(db.objectStoreNames.contains("meta"), "the upgrade must create the meta store");

  // Hello's flush can run before openDB's promise settles (db still null →
  // in-memory path reports 0); the boot flush after the upgrade reports the
  // surviving v1 command — wait for THAT one.
  const pending = await waitFor(
    port.messages,
    (m) => m.type === "pending" && m.count === 1,
    "pending count 1 for the surviving v1 command",
  );
  assert.equal(pending.count, 1, "the v1 pending command must be counted after the upgrade");
  assert.ok(
    !port.messages.some((m) => m.type === "dead"),
    "the v1 pending command must not be evicted by the upgrade boot",
  );
  assert.ok(
    db.stores.get("commands").data.has("legacy-cmd"),
    "the v1 pending command record must survive the upgrade",
  );
});

test("fresh boot creates the full v2 schema (all three stores, no upgrade path)", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  const db = worker.idb.databases.get("cqrshtmx-sync");
  assert.equal(db.version, 2, "a fresh database is created at DB_VERSION 2");
  for (const storeName of ["commands", "events", "meta"]) {
    assert.ok(
      db.objectStoreNames.contains(storeName),
      "a fresh DB must create the " + storeName + " store",
    );
  }
  const pending = await waitFor(port.messages, (m) => m.type === "pending", "pending count");
  assert.equal(pending.count, 0, "a fresh DB has an empty queue");
});

// ---------------------------------------------------------------------------
// Cursor monotonicity (v1.6.0): the persisted cursor must never move
// backwards on the same backend (ULID cursors order lexicographically); a
// backendId reset legitimately rewinds it (new backend, new cursor space).
// ---------------------------------------------------------------------------

test("refuses a regressing cursor on the same backend, allows the rewind across a backend reset", async () => {
  const worker = loadWorker();
  const port = worker.connect();
  port.onmessage({ data: { type: "hello", tabId: "tab-1" } });
  await drain();

  const high = "01JDZZZZZZZZZZZZZZZZZZZZZZZ";
  const low = "01JDAAAAAAAAAAAAAAAAAAAAAAZ";
  assert.ok(low < high, "fixture sanity: the low cursor must sort below the high one");

  port.onmessage({
    data: {
      type: "cache-events",
      cursor: high,
      backendId: "backend-A",
      events: [{ eventId: "e1", type: "Note.created" }],
    },
  });
  await waitFor(port.messages, (m) => m.type === "sync:events", "first cache write");

  // A racing tab delivers a STALE response: lower cursor, same backend.
  port.onmessage({
    data: {
      type: "cache-events",
      cursor: low,
      backendId: "backend-A",
      events: [{ eventId: "e0", type: "Note.created" }],
    },
  });
  await drain();

  const broadcasts = port.messages.filter((m) => m.type === "sync:events");
  assert.equal(broadcasts.length, 2, "the refused write must still broadcast (honest count)");
  assert.equal(
    broadcasts[1].cursor,
    high,
    "the broadcast must carry the KEPT cursor, not the regressed one",
  );

  const state = await askWorker(worker, { type: "get-state" }, "sync-state");
  assert.equal(state.cursor, high, "the persisted cursor must NOT regress");
  assert.ok(
    worker.warns.some((w) => w.indexOf("cursor regression refused") !== -1),
    "the pinned regression log line must fire: " + JSON.stringify(worker.warns),
  );

  // The stale batch's events still land (idempotent puts by eventId).
  const read = await askWorker(worker, { type: "get-events" }, "events");
  assert.equal(
    read.events.map((e) => e.eventId).join(","),
    "e0,e1",
    "the stale batch's events must still be cached",
  );

  // A backendId reset legitimately rewinds the cursor (exempt from the guard).
  port.onmessage({
    data: { type: "cache-events", cursor: low, backendId: "backend-B", events: [] },
  });
  await waitFor(port.messages, (m) => m.type === "sync:reset", "reset on backend change");
  const after = await askWorker(worker, { type: "get-state" }, "sync-state");
  assert.equal(after.cursor, low, "a backend reset must be allowed to rewind the cursor");
});
