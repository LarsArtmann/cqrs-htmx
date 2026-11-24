import { test, expect, type Page } from "@playwright/test";

/**
 * T17 client pack A (F129): chromium smoke for the client-side interactions.
 *
 *  1. Client-side sortable headers (F126): clicking the users table's
 *     "Users" header sorts the LOADED window in place — aria-sort flips, the
 *     row order changes, and NO network request fires.
 *  2. Skeleton honesty (F124/F125): the server-rendered users list carries
 *     the skeleton region hidden (htmx-indicator, opacity 0) — a loaded
 *     page never visibly shows a skeleton — and the shimmer CSS honors
 *     prefers-reduced-motion via the html.admin-reduced-motion mirror.
 *  3. Dirty-form guard attrs (F127) + copy-as-JSON payload (F128) present
 *     on the user detail page.
 *
 * Same recipe as admin-keyboard.spec.ts: SERIAL mode, dev-login beforeEach,
 * chromium (the admin.js behavior suite's project).
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

test.beforeEach(async ({ page }) => {
  await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
  await page.waitForTimeout(500);
});

/** The request CSRF token lives in the layout's meta tag. */
async function csrf(page: Page): Promise<string> {
  const token = await page.evaluate(() =>
    document.querySelector('meta[name="csrf-token"]')?.getAttribute("content"),
  );
  expect(token, "panel layout should render the csrf meta").toBeTruthy();
  return token ?? "";
}

/** Seed one user via the panel's create-user form (CSRF header). */
async function seedUser(page: Page, email: string, displayName: string): Promise<void> {
  const token = await csrf(page);
  const res = await page.request.post(`${ADMIN}/admin/users`, {
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Csrf-Token": token,
    },
    data: `email=${encodeURIComponent(email)}&display_name=${encodeURIComponent(displayName)}`,
  });
  // Already-exists (a previous serial run) is fine — the handler maps the
  // conflict to 400; we only need the row, asserted on the page below.
  expect([200, 400, 409].includes(res.status()), `seed user: ${res.status()}`).toBeTruthy();
}

/**
 * Sort the users table by the email header IN one synchronous evaluate and
 * read {aria, order} back. The demo wires the live-updates refresh (T24),
 * which can swap the whole #users-table region on a background sync event —
 * reading after an async wait would race that swap; the in-place contract
 * is what we pin: same DOM table, new order, zero requests.
 */
function sortAndRead(page: Page) {
  return page.evaluate(() => {
    document
      .querySelector("#users-table button[data-sort-key='email']")
      ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    const th = document.querySelector("#users-table th:has(button[data-sort-key='email'])");
    return {
      aria: th?.getAttribute("aria-sort"),
      order: Array.from(document.querySelectorAll("#users-table tbody tr")).map(
        (r) => r.getAttribute("data-sort-email"),
      ),
    };
  });
}

test("sortable: header click re-orders the loaded window without a round-trip", async ({ page }) => {
  // Seed two clearly-ordered users so the loaded window has rows to sort
  // (the panel's own create-user route; the a11y suite's pattern).
  await seedUser(page, "e2e-clientpack-aaa@acme.dev", "Ada Sort");
  await seedUser(page, "e2e-clientpack-zzz@acme.dev", "Zoe Sort");
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const header = page.locator("#users-table th:has(button[data-sort-key='email'])");
  await expect(header).toHaveAttribute("aria-sort", /^(none|ascending)$/);

  // Count requests while sorting: the contract is NO server round-trip.
  let requests = 0;
  page.on("request", (req) => {
    if (req.url().includes("/admin/users")) requests++;
  });

  const asc = await sortAndRead(page);
  expect(requests, "client sort must not hit the server").toBe(0);
  expect(asc.aria).toBe("ascending");
  expect(asc.order?.length).toBeGreaterThanOrEqual(2);
  expect(asc.order).toEqual([...(asc.order ?? [])].sort((a, b) => (a ?? "").localeCompare(b ?? "")));

  // Second click: descending — the order MUST now be the reverse, which
  // proves the rows were re-ordered in place (the seeded server order is
  // ascending, so the descending toggle is the observable change).
  const desc = await sortAndRead(page);
  expect(desc.aria).toBe("descending");
  expect(desc.order).toEqual([...(desc.order ?? [])].sort((a, b) => (b ?? "").localeCompare(a ?? "")));
  expect(JSON.stringify(desc.order)).not.toBe(JSON.stringify(asc.order));
});

test("skeleton: server render is hidden and reduced-motion freezes the shimmer", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const hidden = await page.evaluate(() => {
    const sk = document.getElementById("users-skeleton");
    if (!sk) return { present: false, opacity: "" };
    return { present: true, opacity: getComputedStyle(sk).opacity };
  });
  expect(hidden.present, "skeleton region must exist in the server render").toBe(true);
  expect(parseFloat(hidden.opacity)).toBeLessThan(0.05);

  // The admin.js reduced-motion mirror (T16 pattern) must freeze the
  // skeleton shimmer: force the class and read the animation.
  const anim = await page.evaluate(() => {
    document.documentElement.classList.add("admin-reduced-motion");
    const bar = document.querySelector(".admin-skeleton");
    return bar ? getComputedStyle(bar).animationName : "missing";
  });
  expect(anim, "admin-reduced-motion must disable the skeleton shimmer").toBe("none");
});

test("detail page: dirty-guard attrs and copy-as-JSON payload present", async ({ page }) => {
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(300);
  await page.locator("#users-table tbody tr a").first().click();
  await page.waitForLoadState("load");
  await page.waitForTimeout(300);

  const guarded = await page.evaluate(
    () => document.querySelectorAll("form[data-dirty-guard]").length,
  );
  expect(guarded).toBeGreaterThanOrEqual(2);

  const payload = await page.evaluate(() =>
    document.querySelector("[data-copy-json]")?.getAttribute("data-copy-json"),
  );
  expect(payload).toBeTruthy();
  const parsed = JSON.parse(
    (payload ?? "").replace(/&#34;/g, '"').replace(/&amp;/g, "&"),
  );
  expect(parsed).toHaveProperty("id");
  expect(parsed).toHaveProperty("email");
  expect(parsed).toHaveProperty("created_at");
});
