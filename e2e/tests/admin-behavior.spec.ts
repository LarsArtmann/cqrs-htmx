import { test, expect } from "@playwright/test";

/**
 * adminui behavior gate (ported from the session-time verify-p3.mjs harness,
 * 2026-09-20). Asserts DOM facts against the admin-demo on :18930 — the
 * screenshot/visual-diff half of the gate stays a session-time activity
 * (adjudicated pixel diffs; see docs/status/2026-09-20_*). Tests are SERIAL:
 * the confirm-modal test deletes a seeded filler user, so every count-based
 * assertion runs first.
 *
 * Coverage: pagination (row counts, aria-current, clamping, search compose),
 * the htmx partial-swap contract, search spinner, the shared confirm modal
 * (open/fill/Esc/Cancel/backdrop/Confirm issues the request), the user-menu
 * dropdown (open/anchor/light-dismiss), the polled stats region, and the
 * favicon 204 fix.
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

test.beforeEach(async ({ page }) => {
  await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
  await page.waitForTimeout(500);
});

test("users pagination: page 1 has 50 of 65 rows with pager", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(page.locator("#users-table tbody tr")).toHaveCount(50);
  const html = await page.content();
  expect(html).toContain("Showing 50 of 65");
  expect(html).toContain("/admin/users?page=2");
});

test("users pagination: page 2 carries aria-current and clamping holds", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users?page=2`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(page.locator("#users-table tbody tr")).toHaveCount(15);
  expect(await page.content()).toContain('aria-current="page"');

  await page.goto(`${ADMIN}/admin/users?page=99`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(page.locator("#users-table tbody tr")).toHaveCount(15);
});

test("users search composes with pagination and scopes to the table", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users?q=team1&page=1`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const table = await page.locator("#users-table").innerText();
  expect(table).toContain("team10@acme.dev");
  expect(table).not.toContain("admin@demo.dev");
});

test("partial-swap contract: fragment carries rows, not the wrapper", async ({ page }) => {
  const partial = await page.evaluate(async () => {
    const r = await fetch("/admin/users?page=2", { headers: { "HX-Request": "true" } });
    return await r.text();
  });
  expect(partial).toContain("<tbody");
  expect(partial).not.toContain('id="users-table"');
  expect(partial).not.toContain("<html");
});

test("search spinner wired via hx-indicator", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const html = await page.content();
  expect(html).toMatch(/<span id="users-search-spinner" class="htmx-indicator[^"]*"/);
  expect(html).toContain('hx-indicator="#users-search-spinner"');
});

test("confirm modal: open, fill, Esc/Cancel/backdrop close, Confirm deletes", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users?q=team60`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const href = await page
    .locator('#users-table a[href*="/admin/users/"]')
    .first()
    .getAttribute("href");
  expect(href).toBeTruthy();
  await page.goto(`${ADMIN}${href}`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const posts: string[] = [];
  page.on("request", (req) => {
    if (req.method() === "POST") posts.push(req.url());
  });

  const dlg = page.locator("#admin-confirm-modal");
  const btn = page.locator('button[data-confirm-title="Delete user"]').first();
  expect(await dlg.getAttribute("open")).toBeNull();

  await btn.click();
  await page.waitForTimeout(300);
  expect(await dlg.getAttribute("open")).not.toBeNull();
  await expect(page.locator("#admin-confirm-modal-title")).toHaveText("Delete user");
  await expect(page.locator("#admin-confirm-body")).toContainText("cannot be undone");

  await page.keyboard.press("Escape");
  await page.waitForTimeout(300);
  expect(await dlg.getAttribute("open")).toBeNull();

  await btn.click();
  await page.waitForTimeout(300);
  await page.getByRole("button", { name: "Cancel" }).click();
  await page.waitForTimeout(300);
  expect(await dlg.getAttribute("open")).toBeNull();
  expect(posts).toHaveLength(0);

  await btn.click();
  await page.waitForTimeout(300);
  await page.mouse.click(5, 5); // backdrop area — dialog element padding zone
  await page.waitForTimeout(300);
  expect(await dlg.getAttribute("open")).toBeNull();

  await btn.click();
  await page.waitForTimeout(300);
  await page.locator("#admin-confirm-ok").click();
  await page.waitForTimeout(900);
  expect(posts.some((u) => u.includes("/delete"))).toBe(true);

  await page.goto(`${ADMIN}/admin/users?q=team60`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(page.locator("#users-table tbody tr")).toHaveCount(0);
});

test("user menu dropdown: opens anchored, light-dismisses", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/`, { waitUntil: "load" });
  await page.waitForTimeout(500);
  const menu = page.locator("#admin-user-menu-menu");
  expect(await menu.evaluate((el) => el.matches(":popover-open"))).toBe(false);

  await page.locator("#admin-user-menu-button").click();
  await page.waitForTimeout(400);
  expect(await menu.evaluate((el) => el.matches(":popover-open"))).toBe(true);
  // Anchored under the header trigger (right side), not the viewport corner.
  const box = await menu.boundingBox();
  expect(box).toBeTruthy();
  if (box) expect(box.x).toBeGreaterThan(500);
  await expect(menu.locator("a[role=menuitem]")).toHaveText("Sign out");

  await page.mouse.click(30, 400);
  await page.waitForTimeout(300);
  expect(await menu.evaluate((el) => el.matches(":popover-open"))).toBe(false);
});

test("dashboard stats: polled region wired to the partial endpoint", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/`, { waitUntil: "load" });
  await page.waitForTimeout(500);
  const html = await page.content();
  expect(html).toContain('hx-get="/admin/partials/stats"');
  expect(html).toContain("every 30s");

  const partial = await page.evaluate(async () => {
    const r = await fetch("/admin/partials/stats", { headers: { "HX-Request": "true" } });
    return { status: r.status, body: await r.text() };
  });
  expect(partial.status).toBe(200);
  expect(partial.body).toContain('hx-get="/admin/partials/stats"');
  expect(partial.body).not.toContain("<html");
});

test("favicon answers 204 without a redirect (no session clobber)", async ({ page }) => {
  const fav = await page.evaluate(async () => {
    const r = await fetch("/favicon.ico", { redirect: "follow" });
    return { status: r.status, redirected: r.redirected };
  });
  expect(fav.status).toBe(204);
  expect(fav.redirected).toBe(false);
});

test("theme toggle: flips .dark on <html>, syncs aria, persists across reload", async ({
  page,
}) => {
  await page.goto(`${ADMIN}/admin/`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const toggle = page.locator("[data-theme-toggle]");
  await expect(toggle).toHaveCount(1);
  await expect(toggle).toHaveAttribute("role", "switch");

  const before = await page.evaluate(() => document.documentElement.classList.contains("dark"));
  await toggle.click();
  await page.waitForTimeout(300);
  const after = await page.evaluate(() => document.documentElement.classList.contains("dark"));
  expect(after).toBe(!before);
  // The switch's checked state follows the applied theme.
  expect(await toggle.getAttribute("aria-checked")).toBe(String(after));

  // The choice is stored (localStorage 'theme') and re-applied pre-paint on reload.
  await page.reload({ waitUntil: "load" });
  await page.waitForTimeout(300);
  const persisted = await page.evaluate(() => document.documentElement.classList.contains("dark"));
  expect(persisted).toBe(after);
});
