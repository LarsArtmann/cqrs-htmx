// Node test suite for the shipped tracking/tracker.js.
//
// The tracker is a browser IIFE with no dependencies, so these tests load the
// REAL file into a vm sandbox whose window IS the sandbox global and drive it
// through injectable fakes: timers, fetch/fetchLater, IntersectionObserver,
// visibility-state entries, localStorage and an event registry. Every test
// asserts the concrete beacon bodies and headers the server would receive.
//
// Run via the Go wrapper or directly:
//   node --test --test-reporter=tap ./tracking/jstests/tracker.test.js
"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { randomFillSync } = require("node:crypto");

const TRACKER_SRC = fs.readFileSync(path.join(__dirname, "..", "tracker.js"), "utf8");
const ULID = /^[0-9A-HJKMNP-TV-Z]{26}$/;
const ORIGIN = "https://example.org";

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/** Deterministic timers the tests drain manually with advance(ms). */
function fakeTimers() {
  let now = 0;
  let seq = 0;
  const pending = new Map(); // id -> { fn, at }
  return {
    setTimeout(fn, ms) {
      const id = ++seq;
      pending.set(id, { fn, at: now + (Number(ms) || 0) });
      return id;
    },
    clearTimeout(id) {
      pending.delete(id);
    },
    /** Run every timer due within `ms`, in due order, including ones scheduled while running. */
    advance(ms) {
      const until = now + ms;
      for (;;) {
        let next = null;
        for (const [id, t] of pending) {
          if (t.at <= until && (!next || t.at < next[1].at)) next = [id, t];
        }
        if (!next) break;
        pending.delete(next[0]);
        now = next[1].at;
        next[1].fn();
      }
      now = until;
    },
    get pending() {
      return pending.size;
    },
  };
}

/** Minimal EventTarget: a listener registry the tests dispatch through. */
function eventRegistry() {
  const listeners = [];
  return {
    add(type, fn, capture) {
      listeners.push({ type, fn, capture });
    },
    remove(type, fn) {
      const i = listeners.findIndex((l) => l.type === type && l.fn === fn);
      if (i >= 0) listeners.splice(i, 1);
    },
    dispatch(type, event) {
      for (const l of listeners.slice()) if (l.type === type) l.fn(event);
    },
    find(type) {
      return listeners.filter((l) => l.type === type);
    },
  };
}

/** A DOM-ish element: attributes, a parent chain and a selector-specific closest(). */
function fakeElement(tag, attrs, parent) {
  const el = {
    nodeType: 1,
    nodeName: tag.toUpperCase(),
    attrs: attrs || {},
    parentNode: parent || null,
    textContent: "Customer lars@example.com bought 3 items",
    getAttribute(name) {
      return name in this.attrs ? this.attrs[name] : null;
    },
    closest(selector) {
      for (let node = this; node; node = node.parentNode) {
        const m = /^\[([\w-]+)(?:="([^"]*)")?\]$/.exec(selector);
        assert.ok(m, "fake closest: unsupported selector " + selector);
        const value = node.attrs[m[1]];
        if (m[2] === undefined ? value !== undefined : value === m[2]) return node;
      }
      return null;
    },
  };
  return el;
}

// ---------------------------------------------------------------------------
// Sandbox factory
// ---------------------------------------------------------------------------

/**
 * Load the real tracker.js into a fresh sandbox.
 * @param {object} [opts] - Injected fakes/switches (see the defaults below).
 */
function loadTracker(opts) {
  const o = opts || {};
  const clock = { t: o.now || 0 }; // o.now: performance.now() at load
  const timers = fakeTimers();
  const store = o.store || new Map();
  const visibilityEntries = [];
  const elements = o.elements || [];
  const observers = [];
  const fetchCalls = [];
  const winEvents = eventRegistry();
  const docEvents = eventRegistry();

  const throwing = () => {
    throw new Error("SecurityError: localStorage is unavailable");
  };
  const localStorage = o.storageThrows
    ? { getItem: throwing, setItem: throwing, removeItem: throwing }
    : {
        getItem: (k) => (store.has(k) ? store.get(k) : null),
        setItem: (k, v) => store.set(k, String(v)),
        removeItem: (k) => store.delete(k),
      };

  class FakeIntersectionObserver {
    constructor(callback, init) {
      this.callback = callback;
      this.init = init;
      this.observed = [];
      this.disconnects = 0;
      observers.push(this);
    }
    observe(el) {
      this.observed.push(el);
    }
    disconnect() {
      this.disconnects++;
      this.observed = [];
    }
  }

  const doc = {
    referrer: o.referrer || "",
    visibilityState: o.visibilityState || "visible",
    currentScript: o.script || null,
    documentElement: { getAttribute: (name) => (name === "data-tracking" && o.htmlTracking) || null },
    querySelector: (selector) =>
      selector === 'meta[name="csrf-token"]' && o.csrf ? { getAttribute: () => o.csrf } : null,
    querySelectorAll: (selector) => {
      assert.equal(selector, "[data-track-impression]");
      return elements.slice();
    },
    addEventListener(type, fn, capture) {
      docEvents.add(type, fn, capture);
    },
    removeEventListener(type, fn) {
      docEvents.remove(type, fn);
    },
  };

  const fetch = o.fetch || ((url, init) => {
    fetchCalls.push({ url, init });
    return Promise.resolve({ ok: true });
  });

  const sandbox = {
    document: doc,
    location: Object.assign({ origin: ORIGIN, pathname: "/", search: "", hash: "" }, o.location),
    performance: {
      now: () => clock.t,
      getEntriesByType: (type) => (type === "visibility-state" ? visibilityEntries.slice() : []),
    },
    fetch,
    localStorage,
    navigator: o.gpc ? { globalPrivacyControl: true } : {},
    setTimeout: timers.setTimeout,
    clearTimeout: timers.clearTimeout,
    addEventListener(type, fn, capture) {
      winEvents.add(type, fn, capture);
    },
    removeEventListener(type, fn) {
      winEvents.remove(type, fn);
    },
    Date,
    Promise,
    AbortController,
    console: { log() {}, warn() {}, error() {} },
  };
  if (!o.noCrypto) sandbox.crypto = { getRandomValues: (arr) => randomFillSync(arr) };
  if (!o.noIntersectionObserver) sandbox.IntersectionObserver = FakeIntersectionObserver;
  if (o.fetchLater) sandbox.fetchLater = o.fetchLater;
  sandbox.window = sandbox;

  const context = vm.createContext(sandbox);
  vm.runInContext(TRACKER_SRC, context, { filename: "tracker.js" });

  return {
    tracker: sandbox.cqrsTracker,
    sandbox,
    store,
    doc,
    timers,
    clock,
    elements,
    observers,
    visibilityEntries,
    fetchCalls,
    location: sandbox.location,
    /** Dispatch on window: popstate, pagehide, pageshow, htmx:* */
    fire: (type, event) => winEvents.dispatch(type, Object.assign({ type }, event)),
    /** Dispatch on document: visibilitychange */
    fireOnDocument: (type, event) => docEvents.dispatch(type, Object.assign({ type }, event)),
    /** Dispatch a capture-phase click with the given target. */
    click: (target) => docEvents.dispatch("click", { type: "click", target }),
    clickListeners: () => docEvents.find("click"),
    /** Let queued microtasks run. */
    microtasks: () => new Promise((resolve) => setImmediate(resolve)),
    /** Shim-path fetch calls as plain host objects with parsed bodies. */
    fetchBeacons: () =>
      fetchCalls.map((c) => ({
        url: c.url,
        method: c.init.method,
        headers: Object.assign({}, c.init.headers),
        keepalive: c.init.keepalive,
        signal: c.init.signal,
        body: JSON.parse(c.init.body),
      })),
  };
}

/** Native-path stub: records every fetchLater call and lets the test flip `activated`. */
function fetchLaterStub(options) {
  const o = options || {};
  const calls = [];
  const fetchLater = (url, init) => {
    const body = JSON.parse(init.body);
    const call = { url, init, body, state: { activated: false } };
    calls.push(call);
    if (o.failAbove !== undefined && body.events.length > o.failAbove) {
      throw new Error("QuotaExceededError");
    }
    return { get activated() { return call.state.activated; } };
  };
  const plain = (call) => ({
    url: call.url,
    method: call.init.method,
    headers: Object.assign({}, call.init.headers),
    keepalive: call.init.keepalive,
    signal: call.init.signal,
    activateAfter: call.init.activateAfter,
    body: call.body,
  });
  return { fetchLater, calls, plain };
}

/** fetch that throws once the beacon carries more than `failAbove` events. */
function quotaFetch(failAbove, calls) {
  return (url, init) => {
    calls.push(init);
    if (JSON.parse(init.body).events.length > failAbove) throw new Error("QuotaExceededError");
    return Promise.resolve({ ok: true });
  };
}

const bodies = (beacons) => beacons.map((b) => b.body);
const eventsOf = (beacons) => beacons.flatMap((b) => b.body.events);

// ---------------------------------------------------------------------------
// Opt-outs
// ---------------------------------------------------------------------------

test("opt-out: global privacy control disables everything permanently", async () => {
  const native = fetchLaterStub();
  const shim = loadTracker({ gpc: true });
  const viaNative = loadTracker({ gpc: true, fetchLater: native.fetchLater });

  for (const env of [shim, viaNative]) {
    assert.equal(env.tracker.version, "1.0.0");
    assert.equal(env.tracker.enabled(), false);
    assert.equal(env.tracker.track("click", { name: "x" }), false);
    env.fire("popstate");
    env.click(fakeElement("button", { "data-track": "buy" }));
    env.tracker.enable(); // GPC is permanent
    assert.equal(env.tracker.enabled(), false);
    env.timers.advance(60000);
    await env.microtasks();
    assert.equal(env.fetchCalls.length, 0, "shim path: no fetch");
  }
  assert.equal(native.calls.length, 0, "native path: no fetchLater");
});

test("opt-out: html data-tracking=off collects nothing and is not persisted", () => {
  const env = loadTracker({ htmlTracking: "off" });
  assert.equal(env.tracker.enabled(), false);
  assert.equal(env.tracker.track("pageview", {}), false);
  env.timers.advance(60000);
  assert.equal(env.fetchCalls.length, 0);
  assert.equal(env.store.has("cqrs-htmx-tracking"), false, "the <html> opt-out is not persisted");

  env.tracker.enable(); // a consent banner may lift it
  assert.equal(env.tracker.enabled(), true);
  env.tracker.track("click", { name: "a" });
  env.timers.advance(5000);
  assert.deepStrictEqual(eventsOf(env.fetchBeacons()).map((e) => e.kind), ["pageview", "click"]);
});

test("opt-out: a persisted off is honoured and enable() resumes", () => {
  const store = new Map([["cqrs-htmx-tracking", "off"]]);
  const env = loadTracker({ store });
  assert.equal(env.tracker.enabled(), false);
  env.timers.advance(60000);
  assert.equal(env.fetchCalls.length, 0);

  env.tracker.enable();
  assert.equal(env.store.has("cqrs-htmx-tracking"), false);
  assert.equal(env.tracker.enabled(), true);
  assert.equal(env.tracker.track("click", { name: "later" }), true);
  env.timers.advance(5000);
  assert.deepStrictEqual(env.fetchBeacons().map((b) => b.body.events), [
    [{ kind: "pageview", path: "/" }, { kind: "click", path: "/", name: "later" }],
  ]);
});

test("disable() cancels the pending shim beacon and clears the queue", () => {
  const env = loadTracker();
  assert.equal(env.timers.pending, 1, "the load beacon waits for the batch window");
  env.tracker.track("click", { name: "a" });

  env.tracker.disable();
  assert.equal(env.tracker.enabled(), false);
  assert.equal(env.store.get("cqrs-htmx-tracking"), "off");
  assert.equal(env.timers.pending, 0, "the batch timer was cleared");
  assert.equal(env.tracker.track("click", { name: "b" }), false);
  env.tracker.flush();
  env.timers.advance(60000);
  assert.equal(env.fetchCalls.length, 0, "nothing is sent after the window is drained");
});

test("disable() aborts the pending native beacon", () => {
  const native = fetchLaterStub();
  const env = loadTracker({ fetchLater: native.fetchLater });
  const signal = native.calls[0].init.signal;
  assert.equal(signal.aborted, false);

  env.tracker.disable();
  assert.equal(signal.aborted, true, "the previous AbortSignal is aborted");
  assert.equal(env.tracker.track("click", { name: "a" }), false);
  assert.equal(native.calls.length, 1, "no new beacon is scheduled while disabled");

  env.tracker.enable();
  assert.equal(native.calls.length, 2, "enable() resumes with a pageview for the current page");
  assert.deepStrictEqual(native.calls[1].body, { events: [{ kind: "pageview", path: "/" }] });
});

// ---------------------------------------------------------------------------
// Wire format
// ---------------------------------------------------------------------------

test("pageview on load carries the wire headers", () => {
  const bare = loadTracker({ referrer: "https://referrer.example/" });
  envDrain(bare);
  assert.deepStrictEqual(bare.fetchBeacons(), [
    {
      url: "/-/track",
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Client-Id": bare.store.get("cqrs-htmx-client-id") },
      keepalive: true,
      signal: bare.fetchBeacons()[0].signal,
      body: { events: [{ kind: "pageview", path: "/", referrer: "https://referrer.example/" }] },
    },
  ]);
  assert.match(bare.fetchBeacons()[0].headers["X-Client-Id"], ULID);
  assert.equal("X-CSRF-Token" in bare.fetchBeacons()[0].headers, false, "no meta -> no CSRF header");

  const withMeta = loadTracker({ csrf: "tok-123" });
  envDrain(withMeta);
  assert.equal(withMeta.fetchBeacons()[0].headers["X-CSRF-Token"], "tok-123");
  assert.equal(typeof withMeta.fetchCalls[0].init.body, "string", "the body is a string, never a stream");
});

function envDrain(env) {
  env.timers.advance(5000);
}

test("endpoint comes from data-track-url and defaults to /-/track", () => {
  const custom = loadTracker({
    script: { getAttribute: (n) => (n === "data-track-url" ? " /custom/track " : null) },
  });
  custom.timers.advance(5000);
  assert.equal(custom.fetchBeacons()[0].url, "/custom/track");

  const native = fetchLaterStub();
  const fallback = loadTracker({ fetchLater: native.fetchLater });
  assert.equal(native.calls[0].url, "/-/track");

  const blank = loadTracker({ script: { getAttribute: () => "" } });
  blank.timers.advance(5000);
  assert.equal(blank.fetchBeacons()[0].url, "/-/track", "an empty attribute falls back to the default");
});

test("paths never include the query string or hash", () => {
  const env = loadTracker({ location: { pathname: "/a/b", search: "?q=1", hash: "#h" } });
  const el = fakeElement("div", { "data-track-impression": "banner" });
  env.elements.push(el);
  env.click(fakeElement("a", { "data-track": "link" }));
  env.observers[0].callback([{ isIntersecting: true, intersectionRatio: 1, target: el }]);

  env.location.pathname = "/a/b";
  env.location.search = "?q=2";
  env.fire("popstate"); // path + query changed -> new page view
  env.timers.advance(5000);

  const events = eventsOf(env.fetchBeacons());
  assert.ok(events.length > 0);
  for (const e of events) assert.equal(e.path, "/a/b");
  assert.deepStrictEqual(events.map((e) => e.kind), [
    "pageview",
    "click",
    "impression",
    "engagement",
    "pageview",
  ]);
  assert.deepEqual(events[4], { kind: "pageview", path: "/a/b", referrer: ORIGIN + "/a/b?q=1" });
  assert.equal(JSON.stringify(events).includes("#h"), false, "the hash is never reported");
});

test("device key is a stable ULID shared through localStorage", () => {
  const store = new Map();
  const first = loadTracker({ store });
  first.timers.advance(5000);
  const id = first.fetchBeacons()[0].headers["X-Client-Id"];
  assert.match(id, ULID);
  assert.equal(store.get("cqrs-htmx-client-id"), id, "same storage key as sync-client.js");

  // Crockford base32 timestamp: the first 10 chars decode to a ms time around now.
  const digits = id
    .slice(0, 10)
    .split("")
    .map((c) => "0123456789ABCDEFGHJKMNPQRSTVWXYZ".indexOf(c).toString(32))
    .join("");
  assert.ok(Math.abs(parseInt(digits, 32) - Date.now()) < 60000, "the ULID timestamp is Date.now()");

  const second = loadTracker({ store });
  second.timers.advance(5000);
  assert.equal(second.fetchBeacons()[0].headers["X-Client-Id"], id, "a second load reuses the key");

  store.set("cqrs-htmx-client-id", "not-a-ulid");
  const repaired = loadTracker({ store });
  repaired.timers.advance(5000);
  assert.match(repaired.fetchBeacons()[0].headers["X-Client-Id"], ULID);
  assert.match(store.get("cqrs-htmx-client-id"), ULID, "an invalid stored value is replaced");

  const blocked = loadTracker({ storageThrows: true });
  assert.doesNotThrow(() => blocked.tracker.disable());
  const noCrypto = loadTracker({ noCrypto: true, store: new Map() });
  assert.match(noCrypto.store.get("cqrs-htmx-client-id"), ULID, "Math.random fallback");
  noCrypto.timers.advance(5000);
  assert.match(noCrypto.fetchBeacons()[0].headers["X-Client-Id"], ULID);
});

// ---------------------------------------------------------------------------
// Shim transport
// ---------------------------------------------------------------------------

test("shim batches three events into one fetch after the window and activates at 20", () => {
  const env = loadTracker();
  env.tracker.track("click", { name: "a" });
  env.tracker.track("click", { name: "b" });
  env.timers.advance(4999);
  assert.equal(env.fetchCalls.length, 0, "nothing before the window elapses");
  env.timers.advance(1);
  assert.equal(env.fetchCalls.length, 1, "ONE fetch for three events");
  assert.deepStrictEqual(eventsOf(env.fetchBeacons()).map((e) => e.name ?? e.kind), ["pageview", "a", "b"]);

  const full = loadTracker();
  for (let i = 1; i <= 19; i++) full.tracker.track("click", { name: "e" + i });
  full.timers.advance(0); // the 20th event activated the beacon immediately
  assert.equal(full.fetchCalls.length, 1);
  assert.equal(eventsOf(full.fetchBeacons()).length, 20);

  const overflow = loadTracker();
  for (let i = 1; i <= 20; i++) overflow.tracker.track("click", { name: "e" + i });
  overflow.timers.advance(0);
  const names = eventsOf(overflow.fetchBeacons()).map((e) => e.name ?? e.kind);
  assert.deepEqual(names, Array.from({ length: 20 }, (_, i) => "e" + (i + 1)), "keep the NEWEST 20");
});

test("shim sends on hidden with keepalive and on the next microtask while hidden", async () => {
  const env = loadTracker();
  env.doc.visibilityState = "hidden";
  env.fireOnDocument("visibilitychange");
  assert.equal(env.fetchCalls.length, 1, "sent synchronously on the hidden transition");
  assert.equal(env.fetchBeacons()[0].keepalive, true);
  assert.equal(env.fetchBeacons()[0].method, "POST");
  assert.equal(env.timers.pending, 0, "the window timer was cleared");

  env.tracker.track("click", { name: "hidden-click" });
  assert.equal(env.fetchCalls.length, 1, "not yet: only after the microtask");
  await env.microtasks();
  assert.equal(env.fetchCalls.length, 2);
  assert.deepStrictEqual(env.fetchBeacons()[1].body, { events: [{ kind: "click", path: "/", name: "hidden-click" }] });

  env.fireOnDocument("visibilitychange"); // listener was removed after sending
  env.timers.advance(60000);
  assert.equal(env.fetchCalls.length, 2);
});

test("flush() sends the queue now and never repeats a sent beacon", () => {
  const env = loadTracker();
  env.tracker.flush();
  env.timers.advance(0);
  assert.equal(env.fetchCalls.length, 1, "flush activates at window 0");
  env.tracker.flush();
  env.timers.advance(60000);
  assert.equal(env.fetchCalls.length, 1, "the same batch is not sent twice");

  env.tracker.track("click", { name: "a" });
  env.tracker.flush();
  env.timers.advance(0);
  assert.equal(env.fetchCalls.length, 2);
  assert.deepEqual(eventsOf(env.fetchBeacons())[1], { kind: "click", path: "/", name: "a" });
});

// ---------------------------------------------------------------------------
// Native fetchLater transport
// ---------------------------------------------------------------------------

test("native fetchLater: activateAfter, abort-then-reschedule with the full queue, reset after activation", async () => {
  const native = fetchLaterStub();
  const env = loadTracker({ fetchLater: native.fetchLater });
  assert.deepStrictEqual(native.plain(native.calls[0]), {
    url: "/-/track",
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Client-Id": env.store.get("cqrs-htmx-client-id") },
    keepalive: undefined,
    signal: native.calls[0].init.signal,
    activateAfter: 5000,
    body: { events: [{ kind: "pageview", path: "/" }] },
  });
  assert.equal(typeof native.calls[0].init.body, "string");

  env.tracker.track("click", { name: "a" });
  assert.equal(native.calls.length, 2);
  assert.equal(native.calls[0].init.signal.aborted, true, "the first signal is aborted");
  assert.equal(native.calls[1].init.signal.aborted, false);
  assert.equal(native.plain(native.calls[1]).activateAfter, 5000);
  assert.deepEqual(
    native.calls[1].body.events.map((e) => e.name ?? e.kind),
    ["pageview", "a"],
    "the new beacon carries BOTH events",
  );

  native.calls[1].state.activated = true; // the browser sent it
  env.tracker.track("click", { name: "b" });
  assert.equal(native.calls.length, 3);
  assert.deepEqual(native.calls[2].body, { events: [{ kind: "click", path: "/", name: "b" }] });

  env.timers.advance(60000);
  await env.microtasks();
  assert.equal(env.fetchCalls.length, 0, "the native path never calls fetch()");
});

test("quota errors retry once with the newest 10 events, then drop the queue", () => {
  // Native path: fetchLater throws synchronously.
  const native = fetchLaterStub({ failAbove: 10 });
  const env = loadTracker({ fetchLater: native.fetchLater });
  for (let i = 1; i <= 10; i++) assert.doesNotThrow(() => env.tracker.track("click", { name: "e" + i }));
  assert.deepEqual(
    native.calls.map((c) => c.body.events.length),
    [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 10],
    "the 11-event beacon threw and was retried with the newest 10",
  );
  assert.deepEqual(
    native.calls[11].body.events.map((e) => e.name),
    Array.from({ length: 10 }, (_, i) => "e" + (i + 1)),
  );

  const failing = fetchLaterStub({ failAbove: 5 });
  const second = loadTracker({ fetchLater: failing.fetchLater });
  for (let i = 1; i <= 5; i++) assert.doesNotThrow(() => second.tracker.track("click", { name: "e" + i }));
  assert.deepEqual(failing.calls.map((c) => c.body.events.length), [1, 2, 3, 4, 5, 6, 6]);
  second.tracker.track("click", { name: "after" });
  assert.deepEqual(failing.calls.at(-1).body, { events: [{ kind: "click", path: "/", name: "after" }] });

  // Shim path: fetch throws inside the timer callback.
  const fetchInits = [];
  const shim = loadTracker({ fetch: quotaFetch(5, fetchInits) });
  for (let i = 1; i <= 5; i++) shim.tracker.track("click", { name: "e" + i });
  assert.doesNotThrow(() => shim.timers.advance(5000));
  assert.deepEqual(
    fetchInits.map((init) => JSON.parse(init.body).events.length),
    [6, 6],
  );
  shim.tracker.track("click", { name: "after" });
  shim.timers.advance(5000);
  assert.deepEqual(JSON.parse(fetchInits.at(-1).body).events, [{ kind: "click", path: "/", name: "after" }]);
});

// ---------------------------------------------------------------------------
// Page views
// ---------------------------------------------------------------------------

test("logical navigation emits pageview + previous engagement, deduplicated", () => {
  const env = loadTracker({ referrer: "https://google.com/", location: { search: "?a=1" }, now: 1000 });

  env.location.pathname = "/items";
  env.location.search = "";
  env.fire("htmx:pushedIntoHistory");
  env.clock.t = 4500;
  env.location.pathname = "/items/42";
  env.fire("popstate");
  env.fire("popstate"); // same URL again
  env.fire("htmx:replacedInHistory");
  env.timers.advance(5000);

  assert.deepStrictEqual(eventsOf(env.fetchBeacons()), [
    { kind: "pageview", path: "/", referrer: "https://google.com/" },
    { kind: "engagement", path: "/", referrer: "https://google.com/" }, // 0 ms visible -> omitted
    { kind: "pageview", path: "/items", referrer: ORIGIN + "/?a=1" },
    { kind: "engagement", path: "/items", referrer: ORIGIN + "/?a=1", visibleMs: 3500 },
    { kind: "pageview", path: "/items/42", referrer: ORIGIN + "/items" },
  ]);
});

test("engagement visible time from visibility-state entries", async () => {
  const env = loadTracker({ now: 2000 }); // page view started at t=2000
  for (const [name, startTime] of [["visible", 1000], ["hidden", 4000], ["visible", 10000]]) {
    env.visibilityEntries.push({ name, startTime });
  }
  env.clock.t = 12000;
  env.fire("pagehide");
  env.timers.advance(5000);
  await env.microtasks(); // pagehide ships on the next microtask

  assert.equal(env.fetchCalls.length, 1);
  assert.equal(eventsOf(env.fetchBeacons()).at(-1).visibleMs, 4000);
});

test("engagement via the accumulator fallback and pagehide flush", async () => {
  const env = loadTracker();
  assert.equal(env.visibilityEntries.length, 0);
  env.doc.visibilityState = "hidden";
  env.clock.t = 3000;
  env.fireOnDocument("visibilitychange"); // hidden at 3000 (this also ships the pageview beacon)
  env.doc.visibilityState = "visible";
  env.clock.t = 8000;
  env.fireOnDocument("visibilitychange");
  env.clock.t = 10000;
  env.fire("pagehide");

  assert.equal(env.fetchCalls.length, 1, "nothing synchronous on pagehide");
  assert.equal(env.timers.pending, 0, "no timer is left to fire");
  env.timers.advance(60000);
  assert.equal(env.fetchCalls.length, 1, "no timer-based send");
  await env.microtasks();
  assert.equal(env.fetchCalls.length, 2, "pagehide flushed on the next microtask");
  assert.deepEqual(eventsOf(env.fetchBeacons())[1], { kind: "engagement", path: "/", visibleMs: 5000 });
});

test("a page loaded hidden counts only the time it was shown", async () => {
  const env = loadTracker({ visibilityState: "hidden" });
  await env.microtasks(); // the load beacon ships immediately while hidden
  env.doc.visibilityState = "visible";
  env.clock.t = 5000;
  env.fireOnDocument("visibilitychange");
  env.clock.t = 7000;
  env.fire("pagehide");
  await env.microtasks();

  assert.deepEqual(
    eventsOf(env.fetchBeacons()).filter((e) => e.kind === "engagement"),
    [{ kind: "engagement", path: "/", visibleMs: 2000 }],
  );
});

test("back/forward cache restore starts a new page view", () => {
  const env = loadTracker();
  env.fire("pagehide");
  env.fire("pageshow", { persisted: false });
  env.tracker.track("click", { name: "a" });
  assert.equal(env.tracker.enabled(), true);
  env.fire("pageshow", { persisted: true });
  env.timers.advance(5000);
  assert.deepEqual(
    eventsOf(env.fetchBeacons()).map((e) => [e.kind, e.path]),
    [
      ["pageview", "/"],
      ["engagement", "/"],
      ["click", "/"],
      ["pageview", "/"],
    ],
  );
});

// ---------------------------------------------------------------------------
// Clicks and impressions
// ---------------------------------------------------------------------------

test("click tracking respects per-element opt-out and never leaks text", () => {
  const env = loadTracker({ referrer: "https://g.example/" });
  const plainZone = fakeElement("section", {});
  const button = fakeElement("button", { id: "checkout", "data-track": "buy" }, plainZone);
  const label = fakeElement("span", {}, button);

  env.click(button);
  assert.deepEqual(eventsOf(env.fetchBeacons()).map((e) => e.name), []);

  env.timers.advance(5000);
  const beacons = env.fetchBeacons();
  assert.deepEqual(beacons[0].body.events, [
    { kind: "pageview", path: "/", referrer: "https://g.example/" },
    { kind: "click", path: "/", referrer: "https://g.example/", name: "buy", target: "button#checkout" },
  ]);
  assert.equal(JSON.stringify(beacons[0].body).includes("lars@example.com"), false, "no text content");

  env.click(label);
  env.timers.advance(5000);
  assert.equal(env.fetchBeacons().at(-1).body.events.at(-1).target, "span", "the event target is described");

  const textNode = { nodeType: 3, nodeName: "#text", data: "Buy" };
  assert.doesNotThrow(() => env.click(textNode));
  env.timers.advance(5000);
  assert.equal(env.fetchCalls.length, 2, "a text node without closest() is ignored");

  const optOut = fakeElement("div", { "data-tracking": "off" });
  env.click(fakeElement("button", { "data-track": "no" }, optOut));
  env.click(fakeElement("button", { "data-track": "no", "data-tracking": "off" }));
  env.timers.advance(5000);
  assert.equal(env.fetchCalls.length, 2, "data-tracking=off on the element or an ancestor suppresses it");
});

test("impressions fire once per page view and rescan after htmx swaps", () => {
  const hero = fakeElement("div", { id: "hero", "data-track-impression": "hero" });
  const env = loadTracker({ referrer: "https://g.example/", elements: [hero] });
  const banner = fakeElement("div", { "data-track-impression": "banner" });
  const quiet = fakeElement("aside", { "data-track-impression": "quiet" }, fakeElement("div", { "data-tracking": "off" }));
  const io = env.observers[0];
  assert.deepEqual(Object.assign({}, io.init), { threshold: 0.5 });
  assert.deepEqual(io.observed, [hero]);

  const report = (el, ratio) => io.callback([{ isIntersecting: ratio > 0, intersectionRatio: ratio, target: el }]);
  report(hero, 0.2);
  env.timers.advance(5000);
  assert.equal(eventsOf(env.fetchBeacons()).length, 1, "below 50 % is not an impression");

  report(hero, 1);
  report(hero, 1); // the same element again
  env.elements.push(banner, quiet);
  env.fire("htmx:afterSettle");
  assert.equal(io.disconnects, 2, "disconnect() before re-observing");
  assert.deepEqual(io.observed, [banner, quiet], "only elements not yet seen in this page view");
  report(banner, 1);
  report(hero, 1); // not re-reported after the rescan
  report(quiet, 1);
  env.timers.advance(5000);

  assert.deepEqual(eventsOf(env.fetchBeacons()).slice(1), [
    { kind: "impression", path: "/", referrer: "https://g.example/", name: "hero", target: "div#hero" },
    { kind: "impression", path: "/", referrer: "https://g.example/", name: "banner", target: "div" },
  ]);

  env.location.pathname = "/next";
  env.fire("htmx:historyRestore"); // new page view: the seen memory is cleared
  report(hero, 1);
  env.timers.advance(5000);
  assert.deepEqual(
    eventsOf(env.fetchBeacons())
      .filter((e) => e.kind === "impression")
      .map((e) => e.path),
    ["/", "/", "/next"],
  );
});

test("no intersection observer means no impressions and no error", () => {
  const env = loadTracker({ noIntersectionObserver: true, elements: [fakeElement("div", { "data-track-impression": "x" })] });
  assert.doesNotThrow(() => env.fire("htmx:afterSettle"));
  env.tracker.track("click", { name: "a" });
  env.timers.advance(5000);
  assert.deepEqual(
    eventsOf(env.fetchBeacons()).map((e) => e.kind),
    ["pageview", "click"],
  );
});

// ---------------------------------------------------------------------------
// Manual API
// ---------------------------------------------------------------------------

test("track() manual API validates kinds and honours overrides", () => {
  const env = loadTracker({ referrer: "https://g.example/" });
  assert.equal(env.tracker.track("click", { name: "x" }), true);
  assert.equal(env.tracker.track("page_load", {}), false, "unknown kind");
  assert.equal(env.tracker.track(undefined, {}), false);
  assert.equal(env.tracker.track("pageview"), true, "props is optional");
  env.timers.advance(5000);
  assert.deepEqual(env.fetchBeacons()[0].body.events.slice(1), [
    { kind: "click", path: "/", referrer: "https://g.example/", name: "x" },
    { kind: "pageview", path: "/", referrer: "https://g.example/" },
  ]);

  const custom = loadTracker();
  custom.tracker.track("impression", { path: "/custom", name: "banner", target: "div#ad", visibleMs: 1234.6 });
  custom.tracker.track("engagement", { path: "https://evil.example/x", visibleMs: -5, referrer: "" });
  custom.tracker.track("click", { name: "x".repeat(300), target: "t".repeat(600) });
  custom.timers.advance(5000);
  const events = custom.fetchBeacons()[0].body.events.slice(1);
  assert.deepEqual(events[0], { kind: "impression", path: "/custom", name: "banner", target: "div#ad", visibleMs: 1235 });
  assert.deepEqual(events[1], { kind: "engagement", path: "/" }, "non-/ path falls back, negative visibleMs omitted");
  assert.equal(events[2].name.length, 200, "name clamped to 200 bytes");
  assert.equal(events[2].target.length, 512, "target clamped to 512 bytes");
});
