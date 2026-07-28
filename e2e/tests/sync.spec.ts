import { test, expect } from '@playwright/test';

// IndexedDB inspection scripts as string expressions. Using strings avoids
// Playwright's TypeScript transformer issues with module-level arrow
// functions that access DOM APIs.

const QUEUE_DEPTH = `(function() {
  return new Promise(function(resolve) {
    try {
      var req = indexedDB.open('cqrshtmx-sync', 1);
      req.onupgradeneeded = function(e) {
        var db = e.target.result;
        if (!db.objectStoreNames.contains('commands')) {
          db.createObjectStore('commands', { keyPath: 'commandId' });
        }
      };
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
      var req = indexedDB.open('cqrshtmx-sync', 1);
      req.onupgradeneeded = function(e) {
        var db = e.target.result;
        if (!db.objectStoreNames.contains('commands')) {
          db.createObjectStore('commands', { keyPath: 'commandId' });
        }
      };
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
async function goOffline(page, context) {
  await context.setOffline(true);
  await page.route('**/api/items', (route) => {
    if (route.request().method() === 'POST') {
      route.abort('failed');
    } else {
      route.continue();
    }
  });
}

async function goOnline(page, context) {
  await page.unroute('**/api/items');
  await context.setOffline(false);
}

// Test 1: Offline command is enqueued and persisted to IndexedDB.
// Verifies: htmx:sendError -> sync-client captures envelope -> SharedWorker
// persists { commandId, envelope, queuedAt, retries } to IndexedDB.

test('offline enqueue persists command envelope to IndexedDB', async ({ page, context }) => {
  await page.goto('/');
  await expect(page.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(500);

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 5000 }).toBe(0);

  await goOffline(page, context);
  await page.waitForTimeout(500);

  await page.fill('input[name="name"]', 'Offline Test Item');
  await page.click('button[type="submit"]');

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(1);

  const entries = await page.evaluate(QUEUE_ENTRIES);
  expect(entries).toHaveLength(1);
  expect(entries[0].commandId).toBeTruthy();
  expect(entries[0].envelope.verb).toBe('POST');
  expect(entries[0].envelope.url).toBe('/api/items');
  expect(entries[0].envelope.headers['X-Command-Id']).toBeTruthy();
  expect(entries[0].envelope.values.name).toBe('Offline Test Item');
  expect(entries[0].retries).toBe(0);
});

// Test 2: Queued command is delivered when connectivity returns.
// Verifies: online event -> SharedWorker flush -> retry -> server processes.

test('online flush delivers queued command to server', async ({ page, context }) => {
  await page.goto('/');
  await expect(page.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(500);

  await goOffline(page, context);
  await page.waitForTimeout(500);

  await page.fill('input[name="name"]', 'Delivered After Reconnect');
  await page.click('button[type="submit"]');
  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(1);

  const before = await page.request.get('/api/debug/items');
  expect((await before.json()).length).toBe(0);

  await goOnline(page, context);

  await expect.poll(async () => {
    const resp = await page.request.get('/api/debug/items');
    const items = await resp.json();
    return items.length;
  }, { timeout: 20000 }).toBeGreaterThanOrEqual(1);

  const resp = await page.request.get('/api/debug/items');
  const items = await resp.json();
  expect(items).toContain('Delivered After Reconnect');
});

// Test 3: Cross-session rebuildAndRetry. A command queued in one tab is
// delivered in a new session via the rebuildAndRetry path, and the queue
// is cleaned up after ACK. This verifies the ADR-0040 IndexedDB persistence
// + rebuildAndRetry cross-session recovery path.

test('cross-session rebuildAndRetry delivers and cleans up', async ({ browser }) => {
  const context = await browser.newContext();

  // Session 1: enqueue while offline, then close.
  const page1 = await context.newPage();
  await page1.goto('/');
  await expect(page1.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page1.waitForTimeout(500);

  await goOffline(page1, context);
  await page1.waitForTimeout(500);

  await page1.fill('input[name="name"]', 'Cross-Session Recovery');
  await page1.click('button[type="submit"]');
  await expect.poll(() => page1.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(1);

  await page1.close();

  // Go online before opening session 2.
  await context.setOffline(false);
  await new Promise((r) => setTimeout(r, 500));

  // Session 2: new page. The worker reads IndexedDB, flushes, and since
  // the originating element is gone, sync-client uses rebuildAndRetry,
  // which calls htmx.ajax() with the original envelope (preserving the
  // X-Command-Id). The server ACKs with the correct commandId, deleting
  // it from IndexedDB.
  const page2 = await context.newPage();
  await page2.goto('/');

  await expect(page2.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );

  await expect.poll(async () => {
    const resp = await page2.request.get('/api/debug/items');
    if (!resp.ok()) return [];
    return await resp.json();
  }, { timeout: 20000 }).toContain('Cross-Session Recovery');

  await expect.poll(() => page2.evaluate(QUEUE_DEPTH), { timeout: 20000 }).toBe(0);

  await context.close();
});

// Test 4: Multiple offline commands are all persisted and delivered.

test('multiple offline commands are queued and delivered on reconnect', async ({ page, context }) => {
  await page.goto('/');
  await expect(page.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(500);

  await goOffline(page, context);
  await page.waitForTimeout(500);

  const names = ['Multi-1', 'Multi-2', 'Multi-3'];
  for (const name of names) {
    await page.fill('input[name="name"]', name);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(300);
  }

  await expect.poll(() => page.evaluate(QUEUE_DEPTH), { timeout: 10000 }).toBe(3);

  await goOnline(page, context);

  await expect.poll(async () => {
    const resp = await page.request.get('/api/debug/items');
    const items = await resp.json();
    return items.filter(function(n) { return names.indexOf(n) >= 0; }).length;
  }, { timeout: 30000 }).toBe(3);
});
