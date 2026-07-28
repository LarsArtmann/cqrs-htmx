import { test, expect } from '@playwright/test';

const queueDepthScript = () => {
  return new Promise((resolve) => {
    const req = indexedDB.open('cqrshtmx-sync', 1);
    req.onsuccess = (e) => resolve(0);
    req.onerror = () => resolve(0);
  });
};

test('offline test', async ({ page, context }) => {
  await page.goto('/');
  await expect(page.locator('[data-sync-status]')).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15_000 },
  );
  await page.waitForTimeout(500);
  await expect.poll(() => page.evaluate(queueDepthScript), { timeout: 5_000 }).toBe(0);
  await context.setOffline(true);
  await page.waitForTimeout(500);
});
