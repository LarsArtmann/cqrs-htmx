import { test, expect, type Page } from "@playwright/test";

/**
 * T19 F144 — bulk select + actions chromium spec.
 *
 * 1. The bulk bar stays hidden until a row is checked; checking one row
 *    shows it with an honest count, and the select-all goes INDETERMINATE
 *    (not checked) for a partial selection.
 * 2. Select-all checks every row and clears the indeterminate state.
 * 3. Submitting an action opens the escalation confirm NAMING THE COUNT,
 *    and confirming renders the per-row outcome panel (the demo server has
 *    no verification sender configured, so the honest panel reports the
 *    per-row refusal — which is exactly the honesty contract).
 * 4. The selection export link carries the checked ids as its ids param.
 *
 * Same recipe as admin-client-pack.spec.ts: SERIAL mode, dev-login
 * beforeEach, chromium. Indeterminate state is asserted via DOM reads —
 * the repo's pinned Playwright has no toBeIndeterminate matcher.
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

test.beforeEach(async ({ page }) => {
  await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
  await page.waitForTimeout(500);
});

async function gotoUsers(page: Page): Promise<void> {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);
}

async function selectAllState(page: Page): Promise<{ checked: boolean; indeterminate: boolean }> {
  return page.evaluate(() => {
    const el = document.querySelector("#users-table input[data-bulk-select-all]") as HTMLInputElement | null;
    return { checked: !!el?.checked, indeterminate: !!el?.indeterminate };
  });
}

test("bulk: bar hidden until a row is checked, then count + indeterminate select-all", async ({ page }) => {
  await gotoUsers(page);

  const bar = page.locator("#users-bulk-bar");
  await expect(bar).toBeHidden();

  const rows = page.locator("#users-table input[data-bulk-row]");
  const rowCount = await rows.count();
  expect(rowCount).toBeGreaterThanOrEqual(1);

  let state = await selectAllState(page);
  expect(state.checked).toBe(false);
  expect(state.indeterminate).toBe(false);

  await rows.first().check();
  await expect(bar).toBeVisible();
  await expect(page.locator("#bulk-count")).toContainText("1 of");
  // Partial selection: the honest three-state.
  state = await selectAllState(page);
  expect(state.indeterminate).toBe(true);
  expect(state.checked).toBe(false);

  // Export link carries exactly the checked id.
  const firstId = await rows.first().getAttribute("value");
  const href = await page.locator("#bulk-export-csv").getAttribute("href");
  expect(href).toContain(`ids=${encodeURIComponent(firstId ?? "")}`);
});

test("bulk: select-all checks every row and clears indeterminate", async ({ page }) => {
  await gotoUsers(page);
  const rows = page.locator("#users-table input[data-bulk-row]");
  const rowCount = await rows.count();

  await page.locator("#users-table input[data-bulk-select-all]").click();
  for (let i = 0; i < rowCount; i++) {
    await expect(rows.nth(i)).toBeChecked();
  }
  const state = await selectAllState(page);
  expect(state.checked).toBe(true);
  expect(state.indeterminate).toBe(false);
  await expect(page.locator("#bulk-count")).toContainText(`${rowCount} of ${rowCount}`);

  // Unchecking one row re-enters the partial state.
  await rows.first().uncheck();
  const partial = await selectAllState(page);
  expect(partial.indeterminate).toBe(true);
});

test("bulk: confirm names the count, outcome panel reports per row", async ({ page }) => {
  await gotoUsers(page);
  const rows = page.locator("#users-table input[data-bulk-row]");
  const rowCount = await rows.count();
  await page.locator("#users-table input[data-bulk-select-all]").click();

  await page.getByRole("button", { name: "Resend verification" }).click();

  // The shared native <dialog> confirm — its body must NAME THE COUNT.
  const dlg = page.locator("#admin-confirm-modal");
  await expect(dlg).toBeVisible();
  await expect(dlg).toContainText(`${rowCount} user`);
  await expect(dlg).toContainText("reported individually");

  await page.locator("#admin-confirm-ok").click();

  // The per-row outcome panel: every selected row gets its own line. The
  // demo server ships no verification sender, so the honest result is
  // per-row refusals with a reason — the panel must report refused rows
  // either way, never a clean success over failures.
  const outcome = page.locator("#bulk-outcome");
  await expect(outcome).toBeVisible();
  await expect(outcome).toContainText("refused");
  // The per-row list is bounded to a readable window (20 lines + the
  // "… and N more" line) while the counts above stay exact — the honest
  // scale contract, asserted at the boundary.
  await expect(outcome.locator("li")).toHaveCount(Math.min(rowCount, 21));
});
