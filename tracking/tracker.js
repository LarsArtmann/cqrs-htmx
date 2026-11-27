// tracker.js: browser-side client signals for the cqrs-htmx tracking endpoint.
//
// Self-contained vanilla-JS tracker, served as a static file and included with
// <script src="/-/tracker.js"></script>. No bundler, no dependencies, no
// inline code (works under a nonce CSP).
//
// WHAT IT DOES: produces the "client signals" the tracking endpoint consumes
// so an app can map activity to a user or device:
//   pageview   : on load and on every LOGICAL navigation (htmx history events
//                + popstate), deduplicated on path + query.
//   engagement : exactly one per page view, when the page view ENDS, with its
//                FOREGROUND (tab visible) time only.
//   click      : one delegated capture-phase listener for [data-track].
//   impression : one IntersectionObserver for [data-track-impression].
//
// TRANSPORT: events are batched (at most 20 per beacon) and sent with
// fetchLater() where available (Chrome/Edge), otherwise through a local shim
// on top of fetch(..., {keepalive: true}) with the same contract.
//
// PRIVACY: nothing is collected while opted out (navigator.globalPrivacyControl,
// <html data-tracking="off">, a persisted "off" in localStorage key
// "cqrs-htmx-tracking", written by cqrsTracker.disable()), nor for a click or
// impression below data-tracking="off". Only location.pathname is reported
// (never query string or hash); element descriptors never contain text, hrefs
// or attribute values other than the id.
//
// PUBLIC API (the only global this defines): window.cqrsTracker
//   version, track(kind, props), flush(), disable(), enable(), enabled()
//
// NO BUILD STEP. No framework. No dependencies.
"use strict";

(function () {
  const VERSION = "1.0.0";
  const DEFAULT_ENDPOINT = "/-/track";

  // Wire limits mirrored from the server: it takes up to 50 events per beacon
  // (413 beyond that) and validates every field length in BYTES, not chars.
  const MAX_BATCH = 20;
  const BATCH_WINDOW_MS = 5000;
  const RETRY_BATCH = 10;
  const MAX_PATH_BYTES = 2048;
  const MAX_REFERRER_BYTES = 2048;
  const MAX_NAME_BYTES = 200;
  const MAX_TARGET_BYTES = 512;
  const MAX_VISIBLE_MS = 86400000;
  const KINDS = ["pageview", "engagement", "click", "impression"];
  const OPT_OUT_KEY = "cqrs-htmx-tracking";
  const CLIENT_ID_KEY = "cqrs-htmx-client-id";

  if (typeof window === "undefined" || typeof document === "undefined") return;

  // --- Configuration -------------------------------------------------------
  // document.currentScript is only set while a classic script executes (null
  // for module or dynamically injected scripts), hence the defensive read.
  const ownTag = document.currentScript;
  const ownAttribute = (name) => (ownTag && typeof ownTag.getAttribute === "function" ? ownTag.getAttribute(name) : "");
  const endpoint = String(ownAttribute("data-track-url") || "").trim() || DEFAULT_ENDPOINT;

  // --- Device key: the SAME key sync/sync-client.js uses (one device, one key) ---
  // Duplicated from sync-client.js on purpose: two standalone no-build scripts
  // cannot share code. This block MUST stay in sync with sync-client.js (same
  // storage key, same ULID format), or the server would see two devices.

  /** Generate a ULID: 26 chars, Crockford Base32, 48-bit ms timestamp + 80 random bits. @returns {string} */
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

  /** Get or create the persistent device key; a per-load id when storage throws. @returns {string} */
  function deviceKey() {
    try {
      let cid = localStorage.getItem(CLIENT_ID_KEY);
      if (!cid || !/^[0-9A-HJKMNP-TV-Z]{26}$/.test(cid)) {
        cid = ulid();
        localStorage.setItem(CLIENT_ID_KEY, cid);
      }
      return cid;
    } catch (e) {
      return ulid();
    }
  }

  const clientId = deviceKey();

  /** localStorage may be missing or throw (private mode): never let that escape. */
  function storage(method, key, value) {
    try {
      return localStorage[method](key, value);
    } catch (e) {
      return null;
    }
  }

  // --- Opt-outs, evaluated once at load, in this order ----------------------
  // GPC is permanent (enable() refuses it); the <html> attribute and the
  // persisted choice are not: enable() resumes. `collecting` is the live switch.
  const gpc = typeof navigator !== "undefined" && navigator.globalPrivacyControl === true;
  const root = document.documentElement;
  const htmlOff = !!(root && root.getAttribute && root.getAttribute("data-tracking") === "off");
  let collecting = !gpc && !htmlOff && storage("getItem", OPT_OUT_KEY) !== "off";

  // --- Clocks: every clock is read from the globals at CALL time ------------
  /** performance.now() (ms since navigation start); Date.now() when unavailable. */
  function now() {
    return typeof performance !== "undefined" && typeof performance.now === "function" ? performance.now() : Date.now();
  }

  const isHidden = () => document.visibilityState === "hidden";

  // --- Page view state -----------------------------------------------------
  // `view` is the page view in progress (null while opted out and after
  // pagehide). Tab switches never end it; they only pause the foreground clock.
  let view = null;
  // Fallback foreground clock for browsers without visibility-state entries:
  // `accumulated` is the visible time of the CURRENT page view so far and
  // `visibleSince` the instant the document became visible (null while hidden).
  let accumulated = 0;
  let visibleSince = null;
  // pagehide counts as "hidden" for beacons: a 0 ms timer never runs during
  // unload, a microtask does.
  let unloading = false;

  /** Length of the [aStart, aEnd) ∩ [bStart, bEnd) interval. */
  const overlap = (aStart, aEnd, bStart, bEnd) => Math.max(0, Math.min(aEnd, bEnd) - Math.max(aStart, bStart));

  /** Foreground (tab visible) milliseconds between two now() instants. */
  function foregroundMs(start, end) {
    // Chrome/Edge record every visibility change on the performance timeline,
    // exact even for changes that happened before this script ran (a page
    // loaded in a background tab counts 0 ms until it is shown). Everywhere
    // else: our own visibilitychange accumulator.
    const entries =
      typeof performance !== "undefined" && typeof performance.getEntriesByType === "function"
        ? performance.getEntriesByType("visibility-state")
        : [];
    if (entries.length === 0) {
      return Math.max(0, Math.round(accumulated + (visibleSince === null ? 0 : end - visibleSince)));
    }
    // Before the first entry the state was the opposite of its name; every
    // entry then holds until the next entry (or `end`).
    let state = entries[0].name === "visible" ? "hidden" : "visible";
    let from = 0;
    let total = 0;
    for (const entry of entries) {
      if (state === "visible") total += overlap(start, end, from, entry.startTime);
      state = entry.name;
      from = entry.startTime;
    }
    return Math.round(total + (state === "visible" ? overlap(start, end, from, end) : 0));
  }

  /** Accumulator fallback: pause the clock while the tab is hidden. */
  function onVisibilityChange() {
    if (!collecting) return;
    if (isHidden()) {
      if (visibleSince !== null) accumulated += now() - visibleSince;
      visibleSince = null;
    } else {
      visibleSince = now();
    }
  }

  // --- Wire events ---------------------------------------------------------
  /** Clamp to at most `max` UTF-8 bytes without splitting a surrogate pair. */
  function clampBytes(str, max) {
    let bytes = 0;
    for (let i = 0; i < str.length; i++) {
      const code = str.charCodeAt(i);
      const size = code < 0x80 ? 1 : code < 0x800 ? 2 : code >= 0xd800 && code <= 0xdbff ? 4 : 3;
      if (bytes + size > max) return str.slice(0, i);
      bytes += size;
      if (size === 4) i++; // the low surrogate belongs to this code point
    }
    return str;
  }

  /** One wire event, clamped to the server's byte limits; empty optional fields are omitted. */
  function makeEvent(kind, path, referrer, name, target, visibleMs) {
    const event = { kind: kind, path: clampBytes(String(path), MAX_PATH_BYTES) };
    if (referrer) event.referrer = clampBytes(String(referrer), MAX_REFERRER_BYTES);
    if (name) event.name = clampBytes(String(name), MAX_NAME_BYTES);
    if (target) event.target = clampBytes(String(target), MAX_TARGET_BYTES);
    if (visibleMs > 0) event.visibleMs = Math.min(visibleMs, MAX_VISIBLE_MS);
    return event;
  }

  // --- Queue + beacons -----------------------------------------------------
  // Every schedule aborts the pending beacon and queues a fresh one carrying
  // the FULL queue (capped to the newest MAX_BATCH events), as the
  // batch-analytics guide prescribes: the browser then owns delivery, even if
  // the page goes away before the window elapses.
  const queue = [];
  let beacon = null; // { controller, result, attempt }

  const sent = (b) => !!(b && b.result && b.result.activated);
  const pendingEvents = () => (queue.length > MAX_BATCH ? queue.slice(queue.length - MAX_BATCH) : queue);
  // Without AbortController a pending beacon cannot be revoked: the stale
  // (smaller, still valid) payload is then sent in addition to the new one.
  const newController = () =>
    typeof AbortController === "function" ? new AbortController() : { signal: undefined, abort() {} };

  function resetQueue() {
    queue.length = 0;
    beacon = null;
  }

  function cancelBeacon() {
    if (beacon) beacon.controller.abort();
    beacon = null;
  }

  /** Request headers: JSON, the device key, and the CSRF token when the app exposes one. */
  function beaconHeaders() {
    const headers = { "Content-Type": "application/json", "X-Client-Id": clientId };
    const meta = document.querySelector && document.querySelector('meta[name="csrf-token"]');
    const csrf = meta && meta.getAttribute("content");
    if (csrf) headers["X-CSRF-Token"] = csrf;
    return headers;
  }

  /**
   * Abort the pending beacon and queue a new one for the current queue.
   * @param {number} windowMs - activateAfter: the batch window, 0 to send now.
   * @param {number} attempt - 0 for the first try, 1 for the quota retry.
   */
  function schedule(windowMs, attempt) {
    if (!collecting || queue.length === 0) return;
    cancelBeacon();
    const controller = newController();
    const record = { controller: controller, result: null, attempt: attempt };
    beacon = record;
    const init = {
      method: "POST",
      headers: beaconHeaders(),
      body: JSON.stringify({ events: pendingEvents() }), // a string: streams are rejected
      activateAfter: windowMs,
      signal: controller.signal,
    };
    try {
      record.result =
        typeof globalThis.fetchLater === "function"
          ? globalThis.fetchLater(endpoint, init)
          : shimFetchLater(endpoint, init, () => sendFailed(record));
    } catch (err) {
      sendFailed(record);
    }
  }

  /**
   * The beacon call threw (fetchLater has a ~64 KiB per-origin quota): keep
   * the newest RETRY_BATCH events and retry once (attempt 1) at window 0; if
   * that fails too, drop the queue. The error never escapes to the page.
   */
  function sendFailed(failed) {
    if (beacon !== failed) return; // superseded by a newer schedule or abort
    if (failed.attempt === 0) {
      queue.splice(0, Math.max(0, queue.length - RETRY_BATCH));
      schedule(0, 1);
    } else {
      queue.length = 0;
      cancelBeacon();
    }
  }

  /**
   * Local stand-in for fetchLater() (Firefox, Safari), modelled on the spec's
   * polyfill but kept private, never installed on globalThis. Sends with
   * fetch(..., {keepalive: true}) when the activateAfter timer fires or the
   * document becomes hidden, and on the next microtask if it already is.
   * @returns {object} Same shape as the native result (an `activated` getter).
   */
  function shimFetchLater(url, init, onError) {
    let timer = null;
    let activated = false;

    function destroy() {
      document.removeEventListener("visibilitychange", onVisibility);
      clearTimeout(timer);
    }

    function sendNow() {
      destroy();
      if (init.signal && init.signal.aborted) return;
      try {
        // A rejected promise (offline) is not actionable: the events are gone
        // either way, so keep the console quiet.
        if (typeof fetch === "function") Promise.resolve(fetch(url, { ...init, keepalive: true })).catch(() => {});
        activated = true;
      } catch (err) {
        onError();
      }
    }

    function onVisibility() {
      if (unloading || isHidden()) sendNow();
    }

    if (unloading || isHidden()) {
      // Already hidden: send ASAP, but let synchronous code finish first.
      Promise.resolve().then(sendNow);
    } else {
      document.addEventListener("visibilitychange", onVisibility);
      if (typeof init.activateAfter === "number" && init.activateAfter >= 0) {
        timer = setTimeout(sendNow, init.activateAfter);
      }
    }
    if (init.signal) init.signal.addEventListener("abort", destroy);
    return { get activated() { return activated; } };
  }

  /** Queue one event (and schedule its beacon). Queues nothing while opted out. */
  function enqueue(event) {
    if (!collecting) return false;
    // The previous beacon is on its way out and carries everything queued so
    // far: start a fresh queue (batch-analytics guide, step 3).
    if (sent(beacon)) resetQueue();
    queue.push(event);
    schedule(queue.length >= MAX_BATCH ? 0 : BATCH_WINDOW_MS, 0);
    return true;
  }

  /** Send the queued events now (window 0); no-op on an empty queue or while opted out. */
  function flush() {
    if (!collecting) return;
    if (sent(beacon)) {
      resetQueue(); // already on its way; never send a batch twice
      return;
    }
    schedule(0, 0);
  }

  // --- Page views ----------------------------------------------------------
  /** End the page view in progress and start a new one for the current location. */
  function startPageView(referrer) {
    endPageView(true);
    const t = now();
    accumulated = 0;
    visibleSince = isHidden() ? null : t;
    view = { path: location.pathname, url: location.pathname + location.search, referrer: referrer, startedAt: t };
    seen.clear(); // impressions are counted once PER page view
    scanImpressions();
    enqueue(makeEvent("pageview", view.path, referrer, "", "", 0));
  }

  /** The engagement signal carries the page view's foreground time; emit=false drops it. */
  function endPageView(emit) {
    if (!view) return;
    if (emit) {
      enqueue(makeEvent("engagement", view.path, view.referrer, "", "", foregroundMs(view.startedAt, now())));
    }
    view = null;
  }

  /** Logical navigation (htmx history events, popstate): one page view per distinct path + query. */
  function onNavigation() {
    if (!collecting) return;
    const url = location.pathname + location.search;
    if (view && url === view.url) return; // several events announce one navigation
    // The in-app referrer is the page we came from, as an absolute URL
    // ("null" is what location.origin reports for opaque origins).
    const origin = location.origin;
    startPageView(view && typeof origin === "string" && origin !== "null" ? origin + view.url : "");
  }

  function onHistoryRestore() {
    onNavigation();
    scanImpressions(); // htmx swapped the DOM from its history cache
  }

  function onPageHide() {
    if (!collecting) return;
    unloading = true;
    endPageView(true);
    flush();
  }

  function onPageShow(event) {
    unloading = false;
    // Back/forward cache: the page view ended at pagehide, so the restored
    // page counts as a new one.
    if (collecting && event && event.persisted && !view) startPageView("");
  }

  // --- Clicks --------------------------------------------------------------
  /**
   * Privacy-safe element descriptor: lowercase tag name plus #id. NEVER text
   * content, hrefs or any other attribute value: text may contain personal
   * data. Clamped to 512 bytes by makeEvent. "" for non-element nodes.
   */
  function describe(node) {
    if (!node || node.nodeType !== 1 || typeof node.nodeName !== "string") return "";
    const id = typeof node.getAttribute === "function" ? node.getAttribute("id") : "";
    return node.nodeName.toLowerCase() + (id ? "#" + id : "");
  }

  function onClick(event) {
    if (!collecting || !view) return;
    const target = event.target;
    // event.target is normally an Element, but synthetic events can carry a
    // text node or something else, so use Element.closest defensively. closest()
    // starts at the element itself, so the opt-out check covers it and all ancestors.
    const el = target && typeof target.closest === "function" ? target.closest("[data-track]") : null;
    if (!el || el.closest('[data-tracking="off"]')) return;
    enqueue(makeEvent("click", view.path, view.referrer, el.getAttribute("data-track"), describe(target), 0));
  }

  // --- Impressions ---------------------------------------------------------
  const seen = new Set(); // elements already reported during the current page view
  let observer = null;

  /** Report each element the first time it is at least half visible in this page view. */
  function onImpression(entries) {
    if (!collecting || !view) return;
    for (const entry of entries || []) {
      const el = entry.target;
      if (!el || !entry.isIntersecting || seen.has(el)) continue;
      // The observer also reports the INITIAL state of every observed element,
      // so isIntersecting alone would fire for a 10 % visible element too.
      const ratio =
        typeof entry.intersectionRatio === "number" ? entry.intersectionRatio : entry.isIntersecting ? 1 : 0;
      const optedOut = typeof el.closest === "function" && el.closest('[data-tracking="off"]');
      if (ratio < 0.5 || optedOut) continue;
      seen.add(el);
      enqueue(
        makeEvent("impression", view.path, view.referrer, el.getAttribute("data-track-impression"), describe(el), 0),
      );
    }
  }

  /** (Re-)observe every [data-track-impression] element not yet seen in this page view. */
  function scanImpressions() {
    if (!observer && typeof IntersectionObserver === "function") {
      observer = new IntersectionObserver(onImpression, { threshold: 0.5 });
    }
    if (!observer || typeof document.querySelectorAll !== "function") return;
    observer.disconnect(); // htmx swaps replace DOM: observe from scratch
    for (const el of Array.from(document.querySelectorAll("[data-track-impression]"))) {
      if (!seen.has(el)) observer.observe(el);
    }
  }

  // --- Public API ----------------------------------------------------------
  /**
   * Emit a manual signal; `props` may override path (default: the current
   * pathname), referrer, name, target and visibleMs.
   * @param {string} kind - One of pageview, engagement, click, impression.
   * @param {object} [props] - Optional field overrides.
   * @returns {boolean} Whether the event was queued.
   */
  function track(kind, props) {
    if (!collecting || KINDS.indexOf(kind) === -1) return false;
    const p = props || {};
    const path = typeof p.path === "string" && p.path.charAt(0) === "/" ? p.path : location.pathname;
    const ms = Math.round(Number(p.visibleMs));
    const referrer = p.referrer !== undefined ? p.referrer : view ? view.referrer : "";
    enqueue(makeEvent(kind, path, referrer, p.name, p.target, ms > 0 ? ms : 0));
    return true;
  }

  /** Persist the opt-out, drop everything pending and stop collecting. */
  function disable() {
    collecting = false;
    storage("setItem", OPT_OUT_KEY, "off");
    queue.length = 0;
    cancelBeacon();
    endPageView(false); // no engagement for a withdrawn page view
    if (observer) observer.disconnect();
  }

  /** Remove the persisted opt-out and resume collecting (refused under GPC). */
  function enable() {
    if (gpc) return;
    storage("removeItem", OPT_OUT_KEY);
    if (collecting) return;
    collecting = true;
    startPageView(typeof document.referrer === "string" ? document.referrer : "");
  }

  // --- Wiring --------------------------------------------------------------
  function on(target, type, fn, options) {
    if (target && typeof target.addEventListener === "function") target.addEventListener(type, fn, options);
  }

  // htmx events bubble to window; htmx itself may be absent (popstate still works).
  on(window, "popstate", onNavigation);
  on(window, "htmx:pushedIntoHistory", onNavigation);
  on(window, "htmx:replacedInHistory", onNavigation);
  on(window, "htmx:historyRestore", onHistoryRestore);
  on(window, "htmx:afterSettle", scanImpressions);
  on(window, "pagehide", onPageHide);
  on(window, "pageshow", onPageShow);
  on(document, "click", onClick, true); // capture: ahead of handlers that stop propagation
  on(document, "visibilitychange", onVisibilityChange);

  window.cqrsTracker = {
    version: VERSION,
    track: track,
    flush: flush,
    disable: disable,
    enable: enable,
    enabled: () => collecting,
  };
  if (collecting) startPageView(typeof document.referrer === "string" ? document.referrer : "");
})();
