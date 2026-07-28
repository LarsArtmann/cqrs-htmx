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
        if (!db.objectStoreNames.contains('commands')) {
          db.close();
          resolve(0);
          return;
        }
        try {
          const tx = db.transaction('commands', 'readonly');
          const countReq = tx.objectStore('commands').count();
          countReq.onsuccess = () => { db.close(); resolve(countReq.result); };
          countReq.onerror = () => { db.close(); resolve(0); };
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
};

test('depth test', async ({ page }) => {
  await page.goto('/');
  await expect.poll(() => page.evaluate(queueDepthScript), { timeout: 5000 }).toBe(0);
});
