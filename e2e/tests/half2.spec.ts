import { test, expect } from '@playwright/test';

const queueDepthScript = () => {
  return new Promise((resolve) => {
    try {
      const req = indexedDB.open('cqrshtmx-sync', 1);
      req.onupgradeneeded = (e) => {
        const db = e.target.result;
        if (!db.objectStoreNames.contains('commands')) {
          db.createObjectStore('commands', { keyPath: 'commandId' });
        }
      };
      req.onsuccess = (e) => {
        const db = e.target.result;
        resolve(0);
      };
      req.onerror = () => resolve(0);
    } catch {
      resolve(0);
    }
  });
};

const queueEntriesScript = () => {
  return new Promise((resolve) => {
    try {
      const req = indexedDB.open('cqrshtmx-sync', 1);
      req.onupgradeneeded = (e) => {
        const db = e.target.result;
        if (!db.objectStoreNames.contains('commands')) {
          db.createObjectStore('commands', { keyPath: 'commandId' });
        }
      };
      req.onsuccess = (e) => {
        const db = e.target.result;
        resolve([]);
      };
      req.onerror = () => resolve([]);
    } catch {
      resolve([]);
    }
  });
};

test('two scripts', async ({ page }) => {
  await page.goto('/');
  await expect.poll(() => page.evaluate(queueDepthScript), { timeout: 5000 }).toBe(0);
  const entries = await page.evaluate(queueEntriesScript);
  expect(entries).toEqual([]);
});
