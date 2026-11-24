import { test, expect } from "@playwright/test";

/**
 * T18 F135 + T20 F150: chromium e2e for the adminui command palette and the
 * users-list saved views / display prefs.
 *
 * Palette (T18):
 *  1. Ctrl+K opens the native <dialog> palette (Cmd+K on Mac).
 *  2. Typing fuzzy-filters the open-time DOM index; Enter jumps.
 *  3. "?" toggles the in-dialog shortcut help.
 *
 * Saved views + prefs (T20):
 *  4. Save a named view of the current filter, see it listed, navigate away,
 *     filter differently, re-apply the view (URL navigation restores it).
 *  5. Column visibility + density toggles persist across a reload via the
 *     container-class contract (classes on #users-table, never on the table
 *     the concurrent rows agent owns).
 *
 * Same recipe as admin-keyboard.spec.ts: SERIAL mode, dev-login beforeEach,
 * chromium project, gotcha-66 browser env from the runner.
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

test.beforeEach(async ({ page }) => {
  await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
  await page.waitForTimeout(500);
});

test("palette: Ctrl+K opens the dialog and ? toggles help", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const dialog = page.locator("#admin-palette");
  await expect(dialog).toBeHidden();

  await page.keyboard.press("Control+k");
  await expect(dialog).toBeVisible();

  const input = page.locator("#admin-palette-input");
  await expect(input).toBeFocused();

  // F134: "?" toggles the help section inside the dialog.
  const help = page.locator("#admin-palette-help");
  await expect(help).toBeHidden();
  await page.keyboard.press("?");
  await expect(help).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
});

test("palette: typing filters the index and Enter jumps", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  await page.keyboard.press("Control+k");
  const dialog = page.locator("#admin-palette");
  await expect(dialog).toBeVisible();

  // The open-time index must contain at least the panel pages (sidebar).
  const options = page.locator("[data-palette-option]");
  expect(await options.count()).toBeGreaterThan(0);

  // Fuzzy filter: type a subsequence of "Dashboard" and Enter-jump there.
  await page.keyboard.type("dash");
  await page.waitForTimeout(150);
  const first = options.first();
  await expect(first).toContainText(/Dashboard|dashboard/i);

  // The sidebar's Dashboard entry points at the panel root (/admin/).
  await page.keyboard.press("Enter");
  await page.waitForURL(/\/admin\/($|\?)/);
  await expect(page.locator("#admin-palette")).toBeHidden();
});

test("saved views: save, list, apply after navigating away", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users?filter=unverified`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const bar = page.locator("[data-users-prefs]");
  await expect(bar).toBeVisible();

  await page.locator("[data-prefs-name]").fill("unverified wave");
  await page.locator("[data-prefs-save]").click();
  await expect(page.locator('[data-prefs-apply="unverified wave"]')).toBeVisible();

  // Navigate somewhere else entirely, then apply the view from there.
  await page.goto(`${ADMIN}/admin/dashboard`, { waitUntil: "load" });
  await page.waitForTimeout(300);
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  await page.locator('[data-prefs-apply="unverified wave"]').click();
  await page.waitForURL(/\/admin\/users\?filter=unverified/);
});

test("prefs: column visibility and density persist across reload", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const table = page.locator("#users-table");
  await page.locator('[data-prefs-col="lastseen"]').click();
  await expect(table).toHaveClass(/admin-cols-hide-lastseen/);

  await page.locator("[data-prefs-density]").click();
  await expect(table).toHaveClass(/admin-density-compact/);

  await page.reload({ waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(page.locator("#users-table")).toHaveClass(/admin-cols-hide-lastseen/);
  await expect(page.locator("#users-table")).toHaveClass(/admin-density-compact/);

  // Toggle back so the serial suite leaves a clean default state.
  await page.locator('[data-prefs-col="lastseen"]').click();
  await page.locator("[data-prefs-density]").click();
  await expect(page.locator("#users-table")).not.toHaveClass(/admin-cols-hide-lastseen/);
  await expect(page.locator("#users-table")).not.toHaveClass(/admin-density-compact/);
});
