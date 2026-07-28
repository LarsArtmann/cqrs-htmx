import { test, expect } from '@playwright/test';

const queueDepthScript = () => {
  return new Promise((resolve) => {
    const req = indexedDB.open('cqrshtmx-sync', 1);
    req.onsuccess = (e) => resolve(0);
    req.onerror = () => resolve(0);
  });
};

const queueEntriesScript = () => {
  return new Promise((resolve) => {
    const req = indexedDB.open('cqrshtmx-sync', 1);
    req.onsuccess = (e) => resolve([]);
    req.onerror = () => resolve([]);
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
  await page.fill('input[name="name"]', 'Test');
  await page.click('button[type="submit"]');
  await expect.poll(() => page.evaluate(queueDepthScript), { timeout: 10_000 }).toBe(1);
  const entries = await page.evaluate(queueEntriesScript);
  expect(entries).toHaveLength(1);
});
