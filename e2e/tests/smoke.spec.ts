import { test, expect } from '@playwright/test';

test('smoke: browser launches and page loads', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Items');
});
