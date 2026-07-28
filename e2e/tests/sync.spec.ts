import { test, expect, type Page, type BrowserContext } from '@playwright/test';

// ============================================================================
// Helpers: IndexedDB inspection from the page context.
// ============================================================================

/**
 * Read the number of queued commands from the SharedWorker's IndexedDB
 * (database "cqrshtmx-sync", object store "commands"). Returns 0 if the
 * database or store does not exist yet.
 */
async function queueDepth(page: Page): Promise<number> {
  return page.evaluate(async () => {
    return new Promise<number>((resolve) => {
      try {
        const req = indexedDB.open('cqrshtmx-sync', 1);
        req.onupgradeneeded = (e: Event) => {
          const db = (e.target as IDBOpenDBRequest).result;
          if (!db.objectStoreNames.contains('commands')) {
            db.createObjectStore('commands', { keyPath: 'commandId' });
          }
        };
        req.onsuccess = (e: Event) => {
          const db = (e.target as IDBOpenDBRequest).result;
          if (!db.objectStoreNames.contains('commands')) {
            db.close();
            resolve(0);
            return;
          }
          try {
            const tx = db.transaction('commands', 'readonly');
            const countReq = tx.objectStore('commands').count();
            countReq.onsuccess = () => {
              db.close();
              resolve(countReq.result);
            };
            countReq.onerror = () => {
              db.close();
              resolve(0);
            };
          } catch {
            db.close();
            resolve(0);
          }
        };
        req.onerror = () => resolve(0);
      } catch {
        resolve(0);
      }
    });
  });
}

/**
 * Read all queued command entries from IndexedDB. Each entry has:
 * { commandId, envelope: { verb, url, values, headers }, queuedAt, retries }
 */
async function queueEntries(page: Page): Promise<Array<{
  commandId: string;
  envelope: { verb: string; url: string; values: Record<string, string> | null; headers: Record<string, string> | null };
  queuedAt: number;
  retries: number;
}>> {
  return page.evaluate(async () => {
    return new Promise<Array<any>>((resolve) => {
      try {
        const req = indexedDB.open('cqrshtmx-sync', 1);
        req.onupgradeneeded = (e: Event) => {
          const db = (e.target as IDBOpenDBRequest).result;
          if (!db.objectStoreNames.contains('commands')) {
            db.createObjectStore('commands', { keyPath: 'commandId' });
          }
        };
        req.onsuccess = (e: Event) => {
          const db = (e.target as IDBOpenDBRequest).result;
          if (!db.objectStoreNames.contains('commands')) {
            db.close();
            resolve([]);
            return;
          }
          try {
            const tx = db.transaction('commands', 'readonly');
            const getAll = tx.objectStore('commands').getAll();
            getAll.onsuccess = () => {
              db.close();
              resolve(getAll.result || []);
            };
            getAll.onerror = () => {
              db.close();
              resolve([]);
            };
          } catch {
            db.close();
            resolve([]);
          }
        };
        req.onerror = () => resolve([]);
      } catch {
        resolve([]);
      }
    });
  });
}

/**
 * Fetch the list of items stored on the server (via the debug endpoint).
 */
async function serverItems(page: Page): Promise<string[]> {
  const resp = await page.request.get('/api/debug/items');
  expect(resp.ok()).toBeTruthy();
  return resp.json();
}

/**
 * Wait for the sync client to finish booting: the SSE connection is established
 * and the SharedWorker is connected. We detect this by waiting for the sync
 * indicator to show a connected/idle state.
 */
async function waitForSyncReady(page: Page): Promise<void> {
  await page.goto('/');
  // The sync indicator transitions through states. "Connected" or "ok" means
  // SSE is live. Give it a generous timeout for initial connection.
  await expect(page.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15_000 },
  );
  // Give the SharedWorker a moment to finish booting + reading IndexedDB.
  await page.waitForTimeout(500);
}

// ============================================================================
// Test 1: Offline command is enqueued and persisted to IndexedDB
// ============================================================================

test('offline enqueue persists command envelope to IndexedDB', async ({ page, context }) => {
  await waitForSyncReady(page);

  // Sanity: queue starts empty
  await expect.poll(() => queueDepth(page), { timeout: 5_000 }).toBe(0);

  // Go offline
  await context.setOffline(true);
  // Wait for the offline event to propagate to the SharedWorker
  await page.waitForTimeout(500);

  // Submit the form while offline — HTMX will fire htmx:sendError,
  // which the sync client captures and enqueues to the SharedWorker.
  await page.fill('input[name="name"]', 'Offline Test Item');
  await page.click('button[type="submit"]');

  // The command should be persisted to IndexedDB
  await expect.poll(() => queueDepth(page), { timeout: 10_000 }).toBe(1);

  // Verify the persisted envelope has the expected shape
  const entries = await queueEntries(page);
  expect(entries).toHaveLength(1);
  expect(entries[0].commandId).toBeTruthy();
  expect(entries[0].envelope.verb).toBe('POST');
  expect(entries[0].envelope.url).toBe('/api/items');
  expect(entries[0].envelope.headers?.['X-Command-Id']).toBeTruthy();
  expect(entries[0].envelope.values?.name).toBe('Offline Test Item');
  expect(entries[0].retries).toBe(0);
});

// ============================================================================
// Test 2: Queued command is delivered when connectivity returns
//         (queue → retry → server receives the command)
// ============================================================================

test('online flush delivers queued command to server', async ({ page, context }) => {
  await waitForSyncReady(page);

  // Enqueue a command while offline
  await context.setOffline(true);
  await page.waitForTimeout(500);

  await page.fill('input[name="name"]', 'Delivered After Reconnect');
  await page.click('button[type="submit"]');
  await expect.poll(() => queueDepth(page), { timeout: 10_000 }).toBe(1);

  // Verify server has no items yet
  expect(await serverItems(page)).toHaveLength(0);

  // Go back online — the SharedWorker will flush and the tab will re-issue
  // the command via HTMX.
  await context.setOffline(false);

  // The command should reach the server. We poll the debug endpoint.
  await expect.poll(async () => {
    const items = await serverItems(page);
    return items.length;
  }, { timeout: 20_000 }).toBeGreaterThanOrEqual(1);

  // Verify the item content
  const items = await serverItems(page);
  expect(items).toContain('Delivered After Reconnect');
});

// ============================================================================
// Test 3: Cross-session rebuildAndRetry — command queued in one tab is
//         delivered in a new session via rebuildAndRetry, and the queue is
//         cleaned up after ACK.
// ============================================================================

test('cross-session rebuildAndRetry delivers and cleans up persisted command', async ({ browser }) => {
  // Create a single browser context so IndexedDB persists across pages.
  const context = await browser.newContext();

  // --- Session 1: enqueue while offline, then close ---
  const page1 = await context.newPage();
  await waitForSyncReady(page1);

  await context.setOffline(true);
  await page1.waitForTimeout(500);

  await page1.fill('input[name="name"]', 'Cross-Session Recovery');
  await page1.click('button[type="submit"]');
  await expect.poll(() => queueDepth(page1), { timeout: 10_000 }).toBe(1);

  // Close session 1 — the originating DOM element is gone.
  await page1.close();

  // Go back online before opening the new session
  await context.setOffline(false);
  await new Promise((r) => setTimeout(r, 500));

  // --- Session 2: new page, command recovered via rebuildAndRetry ---
  const page2 = await context.newPage();
  await page2.goto('/');

  // Wait for the sync client to boot and the SSE to connect.
  await expect(page2.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15_000 },
  );

  // The SharedWorker boots, reads IndexedDB, and flushes the persisted
  // command. Since the originating element doesn't exist on this page,
  // sync-client falls through to rebuildAndRetry → htmx.ajax() with the
  // original envelope (including the original X-Command-Id header).
  // The server processes it and broadcasts sync:ack, which deletes the
  // command from IndexedDB.

  // Verify the item reaches the server
  await expect.poll(async () => {
    const resp = await page2.request.get('/api/debug/items');
    if (!resp.ok()) return [];
    const items = await resp.json();
    return items;
  }, { timeout: 20_000 }).toContain('Cross-Session Recovery');

  // Verify the IndexedDB queue is cleaned up after ACK
  await expect.poll(() => queueDepth(page2), { timeout: 20_000 }).toBe(0);

  await context.close();
});

// ============================================================================
// Test 4: Multiple offline commands are all persisted and delivered
// ============================================================================

test('multiple offline commands are queued and delivered on reconnect', async ({ page, context }) => {
  await waitForSyncReady(page);

  await context.setOffline(true);
  await page.waitForTimeout(500);

  // Enqueue three commands
  const names = ['Multi-1', 'Multi-2', 'Multi-3'];
  for (const name of names) {
    await page.fill('input[name="name"]', name);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(200); // small delay between submissions
  }

  // All three should be persisted
  await expect.poll(() => queueDepth(page), { timeout: 10_000 }).toBe(3);

  // Go online
  await context.setOffline(false);

  // All three should reach the server (staggered delivery)
  await expect.poll(async () => {
    const items = await serverItems(page);
    return items.filter((n: string) => names.includes(n)).length;
  }, { timeout: 30_000 }).toBe(3);

  // Cleanup
  await context.close();
});
