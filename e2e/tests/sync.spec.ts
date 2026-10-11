import { test, expect, type Page, type BrowserContext, type Route } from "@playwright/test";

// IndexedDB inspection scripts as string expressions. Using strings avoids
// Playwright's TypeScript transformer issues with module-level arrow
// functions that access DOM APIs.

// Version-agnostic on purpose: the worker owns the DB version (v2 since
// sync assets 1.5.0 added the events+meta stores). Opening with a pinned
// lower version fires VersionError against a live v2 database, which made
// QUEUE_DEPTH resolve 0 forever (found 2026-10-09 against the 1.5.0 worker).

const QUEUE_DEPTH = `(function() {
  return new Promise(function(resolve) {
    try {
      var req = indexedDB.open('cqrshtmx-sync');
      req.onsuccess = function(e) {
        var db = e.target.result;
        if (!db.objectStoreNames.contains('commands')) {
          db.close(); resolve(0); return;
        }
        try {
          var tx = db.transaction('commands', 'readonly');
          var countReq = tx.objectStore('commands').count();
          countReq.onsuccess = function() { db.close(); resolve(countReq.result); };
          countReq.onerror = function() { db.close(); resolve(0); };
        } catch (err) { db.close(); resolve(0); }
      };
      req.onerror = function() { resolve(0); };
    } catch (err) { resolve(0); }
  });
})()`;

const QUEUE_ENTRIES = `(function() {
  return new Promise(function(resolve) {
    try {
      var req = indexedDB.open('cqrshtmx-sync');
      req.onsuccess = function(e) {
        var db = e.target.result;
        if (!db.objectStoreNames.contains('commands')) {
          db.close(); resolve([]); return;
        }
        try {
          var tx = db.transaction('commands', 'readonly');
          var getAll = tx.objectStore('commands').getAll();
          getAll.onsuccess = function() { db.close(); resolve(getAll.result || []); };
          getAll.onerror = function() { db.close(); resolve([]); };
        } catch (err) { db.close(); resolve([]); }
      };
      req.onerror = function() { resolve([]); };
    } catch (err) { resolve([]); }
  });
})()`;

// Offline simulation helper. Uses BOTH context.setOffline (so the
// SharedWorker sees navigator.onLine=false and does not flush) AND route
// interception (so the XHR fails immediately, triggering htmx:sendError).
// Using setOffline alone causes the XHR to hang (Chrome does not abort
// pending XHRs on offline for ~75s TCP timeout).
async function goOffline(page: Page, context: BrowserContext) {
  await context.setOffline(true);
  await page.route("**/api/items", (route: Route) => {
    if (route.request().method() === "POST") {
      route.abort("failed");
    } else {
      route.continue();
    }
  });
}

async function goOnline(page: Page, context: BrowserContext) {
  await page.unroute("**/api/items");
  await context.setOffline(false);
}

type QueueEntry = {
  commandId: string;
  envelope: {
    verb: string;
    url: string;
    headers: Record<string, string>;
    values: Record<string, string>;
  };
  retries: number;
};

// Test 1: Offline command is enqueued and persisted to IndexedDB.
// Verifies: htmx:sendError -> sync-client captures envelope -> SharedWorker
// persists { commandId, envelope, queuedAt, retries } to IndexedDB.

test("offline enqueue persists command envelope to IndexedDB", async ({ page, context }) => {
  await page.goto("/");
  await expect(page.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(500);

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 5000 }).toBe(0);

  await goOffline(page, context);
  await page.waitForTimeout(500);

  await page.fill('#add-form input[name="name"]', "Offline Test Item");
  await page.click('#add-form button[type="submit"]');

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(1);

  const entries = await page.evaluate<QueueEntry[]>(QUEUE_ENTRIES);
  expect(entries).toHaveLength(1);
  expect(entries[0].commandId).toBeTruthy();
  expect(entries[0].envelope.verb).toBe("POST");
  expect(entries[0].envelope.url).toBe("/api/items");
  expect(entries[0].envelope.headers["X-Command-Id"]).toBeTruthy();
  expect(entries[0].envelope.values.name).toBe("Offline Test Item");
  expect(entries[0].retries).toBe(0);
});

// Test 2: Queued command is delivered when connectivity returns.
// Verifies: online event -> SharedWorker flush -> retry -> server processes.

test("online flush delivers queued command to server", async ({ page, context }) => {
  await page.goto("/");
  await expect(page.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(500);

  await goOffline(page, context);
  await page.waitForTimeout(500);

  await page.fill('#add-form input[name="name"]', "Delivered After Reconnect");
  await page.click('#add-form button[type="submit"]');
  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(1);

  const before = await page.request.get("/api/debug/items");
  expect((await before.json()).length).toBe(0);

  await goOnline(page, context);

  await expect
    .poll(
      async () => {
        const resp = await page.request.get("/api/debug/items");
        const items = await resp.json();
        return items.length;
      },
      { timeout: 20000 },
    )
    .toBeGreaterThanOrEqual(1);

  const resp = await page.request.get("/api/debug/items");
  const items = await resp.json();
  expect(items).toContain("Delivered After Reconnect");
});

// Test 3: Cross-session rebuildAndRetry. A command queued in one tab is
// delivered in a new session via the rebuildAndRetry path, and the queue
// is cleaned up after ACK. This verifies the ADR-0040 IndexedDB persistence
// + rebuildAndRetry cross-session recovery path.

test("cross-session rebuildAndRetry delivers and cleans up", async ({ browser }) => {
  const context = await browser.newContext();

  // Session 1: enqueue while offline, then close.
  const page1 = await context.newPage();
  await page1.goto("/");
  await expect(page1.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page1.waitForTimeout(500);

  await goOffline(page1, context);
  await page1.waitForTimeout(500);

  await page1.fill('#add-form input[name="name"]', "Cross-Session Recovery");
  await page1.click('#add-form button[type="submit"]');
  await expect.poll(() => page1.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(1);

  // Close session 1. The worker's round-robin + periodic re-flush will
  // eventually deliver the retry to session 2 (even if the dead port
  // silently swallows the first attempt).
  await page1.close();

  // Go online before opening session 2.
  await context.setOffline(false);
  await new Promise((r) => setTimeout(r, 1000));

  // Session 2: new page. The worker reads IndexedDB, flushes via round-robin,
  // and since the originating element is gone, sync-client uses rebuildAndRetry,
  // which calls htmx.ajax() with the original envelope (preserving the
  // X-Command-Id). The server processes the request and the command is ACK'd.
  const page2 = await context.newPage();
  await page2.goto("/");
  await page2.waitForTimeout(1000);

  // Verify the command was delivered to the server (the retry may take a
  // few seconds via round-robin + periodic re-flush if the first attempt
  // hits a dead port from the closed session 1).
  await expect
    .poll(
      async () => {
        const resp = await page2.request.get("/api/debug/items");
        if (!resp.ok()) return [];
        return await resp.json();
      },
      { timeout: 20000 },
    )
    .toContain("Cross-Session Recovery");

  // Verify queue is cleaned up after ACK
  await expect.poll(() => page2.evaluate(QUEUE_DEPTH), { timeout: 20000 }).toBe(0);

  await context.close();
});

// Test 4: Multiple offline commands are all persisted and delivered.

test("multiple offline commands are queued and delivered on reconnect", async ({
  page,
  context,
}) => {
  await page.goto("/");
  await expect(page.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(500);

  await goOffline(page, context);
  await page.waitForTimeout(500);

  const names = ["Multi-1", "Multi-2", "Multi-3"];
  for (const name of names) {
    await page.fill('#add-form input[name="name"]', name);
    await page.click('#add-form button[type="submit"]');
    await page.waitForTimeout(300);
  }

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(3);

  await goOnline(page, context);

  await expect
    .poll(
      async () => {
        const resp = await page.request.get("/api/debug/items");
        const items = (await resp.json()) as string[];
        return items.filter(function (n: string) {
          return names.indexOf(n) >= 0;
        }).length;
      },
      { timeout: 30000 },
    )
    .toBe(3);
});

// Test 5 (ADR-0056 batch push): commands stamped with
// data-sync-command-type flush as ONE retry-batch POST /sync/push on
// reconnect — not per-command HTMX replays. Verifies: single request,
// all commands in the batch, per-command confirmed outcomes drain the
// queue, and delivery is proven server-side via the debug endpoint.

test("batched offline commands flush as ONE /sync/push request", async ({ page, context }) => {
  await page.goto("/");
  await expect(page.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(500);

  await context.setOffline(true);
  await page.route("**/api/sync-items", (route: Route) => route.abort("failed"));

  const names = ["Batch-1", "Batch-2", "Batch-3"];
  for (const name of names) {
    await page.fill('#sync-add-form input[name="name"]', name);
    await page.click('#sync-add-form button[type="submit"]');
    await page.waitForTimeout(300);
  }

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(3);

  // Arm the push counter BEFORE reconnecting: flushes only fire once the
  // worker sees the online event, so nothing is counted while offline.
  let pushCalls = 0;
  let batchedCommandCount = 0;
  await page.route("**/sync/push", async (route: Route) => {
    if (route.request().method() === "POST") {
      pushCalls++;
      const body = route.request().postDataJSON() as { commands?: unknown[] };
      if (body && Array.isArray(body.commands)) {
        batchedCommandCount = Math.max(batchedCommandCount, body.commands.length);
      }
    }
    await route.continue();
  });

  await page.unroute("**/api/sync-items");
  await context.setOffline(false);

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 30000 }).toBe(0);

  await expect
    .poll(
      async () => {
        const resp = await page.request.get("/api/debug/items");
        const items = (await resp.json()) as string[];
        return names.filter(function (n: string) {
          return items.indexOf(n) >= 0;
        }).length;
      },
      { timeout: 30000 },
    )
    .toBe(3);

  expect(pushCalls).toBe(1);
  expect(batchedCommandCount).toBe(3);
});

// Typed window.cqrsSync access (no global declaration file for specs).
type SyncPublicAPI = {
  getEvents: () => Promise<{ events: Array<{ type?: string }>; cursor: string; backendId: string }>;
  pull: () => Promise<unknown>;
  version: string;
};

function cqrsSync(): SyncPublicAPI {
  return (window as unknown as { cqrsSync: SyncPublicAPI }).cqrsSync;
}

const CACHED_EVENTS = `(async function() {
  const out = await window.cqrsSync.getEvents();
  return { count: out.events.length, backendId: out.backendId };
})()`;

// Test 6 (ADR-0056 offline reads): events pulled while online stay
// readable through window.cqrsSync.getEvents() with the network cut —
// the cached feed is the offline read surface for consumer UIs.

test("offline reads serve cached events via window.cqrsSync.getEvents()", async ({
  page,
  context,
}) => {
  await page.goto("/");
  await expect(page.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );

  // First pull populates the cache from the seeded journal.
  const online = await page.evaluate(cqrsSync);
  expect(online.events.length).toBeGreaterThanOrEqual(2);
  expect(online.backendId).toBe("e2e-backend-A");

  await goOffline(page, context);
  await page.waitForTimeout(500);

  const offline = await page.evaluate(cqrsSync);
  expect(offline.events.length).toBeGreaterThanOrEqual(2);
  expect(offline.events[offline.events.length - 1].type).toBe("Item.synced");
  expect(offline.backendId).toBe("e2e-backend-A");

  await goOnline(page, context);
});

// Test 7 (ADR-0056 sync:reset): when the pull handler's backendId
// changes (backend rebuild), the client wipes the event CACHE but KEEPS
// the command queue — pending work is never lost to a cache reset.

test("sync:reset on backendId change clears the cache but keeps the queue", async ({
  page,
  context,
}) => {
  await page.goto("/");
  await expect(page.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );

  const initial = await page.evaluate(cqrsSync);
  expect(initial.events.length).toBeGreaterThanOrEqual(2);

  // Rotate the backend identity, then force a pull: the response's
  // backendId no longer matches the persisted one, so the worker wipes
  // the cache, stores the (empty, cursor is already past the journal)
  // batch, and broadcasts sync:reset to every tab.
  const rotate = await page.request.post("/api/debug/rotate-backend");
  expect(rotate.status()).toBe(204);
  await page.evaluate(async () => await window.cqrsSync.pull());

  await expect
    .poll(() => page.evaluate(CACHED_EVENTS), { timeout: 10000 })
    .toEqual({ count: 0, backendId: "e2e-backend-B" });

  // Enqueue a command offline AFTER the reset: the queue must accept and
  // keep it (cache reset never touches pending commands).
  await goOffline(page, context);
  await page.waitForTimeout(300);
  await page.fill('#add-form input[name="name"]', "Queued-After-Reset");
  await page.click('#add-form button[type="submit"]');

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(1);

  const afterQueue = await page.evaluate(CACHED_EVENTS);
  expect(afterQueue).toEqual({ count: 0, backendId: "e2e-backend-B" });

  await goOnline(page, context);
});
