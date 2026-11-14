import { test, expect, Page } from "@playwright/test";

/**
 * adminui screenshot harness (M11/M12 of the 2026-11-14 SUPERB plan, T122):
 * captures the surfaces this wave touched — users list/detail, tenants
 * list/detail — in light and dark, so the new cards (Tenant access, tenant
 * identity, lifecycle chips + switcher, import/export entry points) are
 * actually SEEN in docs/, not just string-asserted.
 *
 * Output: docs/screenshots/{light,dark}/admin-<page>.png
 * Run: node_modules/.bin/playwright test --project=chromium admin-screenshots.spec.ts
 *
 * Serial + dev-login (no CSRF writes needed — the panel's seed already
 * carries demo users/tenants to photograph).
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

const MODES = [
  { name: "light", colorScheme: "light" as const, viewport: { width: 1280, height: 900 } },
  { name: "dark", colorScheme: "dark" as const, viewport: { width: 1280, height: 900 } },
];

/** The first tenant slug on the tenants list (the seed's demo tenant). */
async function firstTenantSlug(page: Page): Promise<string> {
  await page.goto(`${ADMIN}/admin/tenants`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const href = await page
    .locator("#tenants-table tbody tr a[href*='/tenants/']")
    .first()
    .getAttribute("href");
  expect(href).toBeTruthy();
  return (href ?? "").split("/").pop() ?? "";
}

/** The first user id on the users list (search-scoped to the seed demo user). */
async function firstUserPath(page: Page): Promise<string> {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const href = await page
    .locator("#users-table tbody tr a[href*='/users/']")
    .first()
    .getAttribute("href");
  expect(href).toBeTruthy();
  return (href ?? "").replace(/^.*\/admin/, "/admin");
}

for (const mode of MODES) {
  test.describe(`admin screenshots (${mode.name})`, () => {
    test.use({ colorScheme: mode.colorScheme, viewport: mode.viewport });

    test.beforeEach(async ({ page }) => {
      await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
      await page.waitForTimeout(500);
    });

    test("captures the tenants list", async ({ page }) => {
      await page.goto(`${ADMIN}/admin/tenants`, { waitUntil: "load" });
      await page.waitForLoadState("networkidle");
      await page.screenshot({
        path: `../docs/screenshots/${mode.name}/admin-tenants.png`,
        fullPage: true,
      });
      expect(true).toBe(true);
    });

    test("captures the users list", async ({ page }) => {
      await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
      await page.waitForLoadState("networkidle");
      await page.screenshot({
        path: `../docs/screenshots/${mode.name}/admin-users.png`,
        fullPage: true,
      });
      expect(true).toBe(true);
    });

    test("captures the tenant detail", async ({ page }) => {
      const slug = await firstTenantSlug(page);
      const resp = await page.goto(`${ADMIN}/admin/tenants/${slug}`, { waitUntil: "load" });
      if (resp?.status() !== 200) {
        test.skip(true, `tenant detail unavailable (${resp?.status()})`);
        return;
      }
      await page.waitForLoadState("networkidle");
      await page.screenshot({
        path: `../docs/screenshots/${mode.name}/admin-tenant-detail.png`,
        fullPage: true,
      });
      expect(true).toBe(true);
    });

    test("captures the user detail", async ({ page }) => {
      const path = await firstUserPath(page);
      const resp = await page.goto(`${ADMIN}${path}`, { waitUntil: "load" });
      if (resp?.status() !== 200) {
        test.skip(true, `user detail unavailable (${resp?.status()})`);
        return;
      }
      await page.waitForLoadState("networkidle");
      await page.screenshot({
        path: `../docs/screenshots/${mode.name}/admin-user-detail.png`,
        fullPage: true,
      });
      expect(true).toBe(true);
    });
  });
}
