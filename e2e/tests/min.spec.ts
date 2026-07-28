import { test, expect } from '@playwright/test';

test('minimal', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toHaveText('Items');
});
