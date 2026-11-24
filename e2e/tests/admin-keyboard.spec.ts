import { test, expect } from "@playwright/test";

/**
 * T15 F112: keyboard-path spec for the adminui panel, extending the
 * keyboard tests in admin-a11y.spec.ts (which pin the grant-form Tab ring)
 * with the admin.js list-navigation contract (F111) and the post-swap
 * focus-restore contract (F110):
 *
 *  1. "/" focuses the list search; typing is never hijacked while the user
 *     is already in a control.
 *  2. Escape clears the search and re-runs the (htmx) search.
 *  3. ArrowDown/ArrowUp traverse the visible list rows (rows become
 *     programmatic focus targets with tabindex="-1", never Tab stops).
 *  4. After an HTMX swap replaces the focused trigger's region (the audit
 *     date form lives inside #audit-panel), focus is restored to the fresh
 *     DOM's matching data-focus-restore element.
 *
 * Same recipe as admin-a11y.spec.ts: SERIAL mode, dev-login beforeEach,
 * chromium (the admin.js behavior suite's project).
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

test.beforeEach(async ({ page }) => {
  await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
  await page.waitForTimeout(500);
});

test("keyboard: '/' focuses the list search from the page body", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  await page.keyboard.press("/");
  const focusedIsSearch = await page.evaluate(
    () => (document.activeElement as HTMLInputElement | null)?.type === "search",
  );
  expect(focusedIsSearch, "pressing '/' must move focus to the list search").toBe(true);

  // Typing "/" INSIDE a control must not re-hijack focus: focus stays put.
  await page.keyboard.type("a//");
  const stillSearch = await page.evaluate(
    () => (document.activeElement as HTMLInputElement | null)?.type === "search",
  );
  expect(stillSearch).toBe(true);
});

test("keyboard: Escape clears the list search and re-runs it", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const search = page.locator("input[type='search']").first();
  await search.fill("ada");
  await search.focus();
  await page.keyboard.press("Escape");

  await expect(search).toHaveValue("", { timeout: 3000 });
});

test("keyboard: ArrowDown/ArrowUp traverse the users list rows", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const rowCount = await page.locator("#users-table tbody tr").count();
  test.skip(rowCount === 0, "empty seed: no rows to traverse");

  // From the search, ArrowDown enters the list (first row)…
  const search = page.locator("input[type='search']").first();
  await search.focus();
  await page.keyboard.press("ArrowDown");
  let activeTag = await page.evaluate(() => document.activeElement?.tagName);
  expect(activeTag, "ArrowDown from the search must land on the first row").toBe("TR");

  // …ArrowDown walks down, ArrowUp walks back up.
  await page.keyboard.press("ArrowDown");
  activeTag = await page.evaluate(() => document.activeElement?.tagName);
  expect(activeTag).toBe("TR");
  await page.keyboard.press("ArrowUp");
  await page.keyboard.press("ArrowUp");
  activeTag = await page.evaluate(() => document.activeElement?.tagName);
  expect(activeTag).toBe("TR");

  // Rows are programmatic focus targets only: they carry tabindex="-1"
  // (set by admin.js on first focus), so the Tab ring is unchanged.
  const rowTabindex = await page.evaluate(() =>
    (document.querySelector("#users-table tbody tr:focus") as HTMLElement | null)?.getAttribute(
      "tabindex",
    ),
  );
  expect(rowTabindex).toBe("-1");
});

test("focus restore: the audit Apply-dates button regains focus after the panel swap", async ({
  page,
}) => {
  await page.goto(`${ADMIN}/admin/audit`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  // The button is the marked trigger (data-focus-restore) AND sits inside
  // the swapped #audit-panel region, so the swap destroys the focused node.
  const apply = page.getByRole("button", { name: "Apply dates" });
  await apply.click();

  // htmx:afterSwap → admin.js re-queries the fresh DOM for the same
  // data-focus-restore value and refocuses it (gotcha 57: document-level
  // delegation, per-event re-query — element-bound JS died with the old DOM).
  await expect
    .poll(
      async () =>
        await page.evaluate(() => {
          const el = document.activeElement as HTMLElement | null;
          return `${el?.tagName ?? ""}:${el?.getAttribute("data-focus-restore") ?? ""}`;
        }),
      { timeout: 5000 },
    )
    .toMatch(/BUTTON:audit-apply|INPUT:audit-apply/);
});
