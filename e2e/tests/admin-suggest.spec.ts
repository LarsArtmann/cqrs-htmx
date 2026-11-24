import { test, expect, Page } from "@playwright/test";

/**
 * adminui type-ahead directory suggest (T08 of the 2026-11-24 SUPERB wave).
 * Same recipe as admin-tenant-access.spec.ts: SERIAL tests, shared in-memory
 * seed on :18930, dev-login before each test, CSRF from the page's meta tag.
 *
 * Coverage: the endpoint contract (JSON array, cap, empty-query refusal) and
 * the combobox pick path — type, arrow, Enter fills the field, submit joins
 * the member (gotcha-66 browser env: E2E_BROWSER_PATH/E2E_BROWSER_LIBS).
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

test.beforeEach(async ({ page }) => {
  await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
  await page.waitForTimeout(500);
});

/** Seed a user through the panel; returns the user detail path. */
async function createUser(page: Page, email: string, name: string): Promise<string> {
  const token = await page.evaluate(() =>
    document.querySelector('meta[name="csrf-token"]')?.getAttribute("content"),
  );
  const res = await page.request.post(`${ADMIN}/admin/users`, {
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Csrf-Token": token ?? "",
    },
    data: `email=${encodeURIComponent(email)}&display_name=${encodeURIComponent(name)}`,
  });
  expect(res.ok, `create user ${email}: ${res.status}`).toBeTruthy();
  return new URL(res.url()).pathname;
}

/** Seed a tenant through the panel; returns the tenant detail path. */
async function createTenant(page: Page, name: string, display: string): Promise<string> {
  const token = await page.evaluate(() =>
    document.querySelector('meta[name="csrf-token"]')?.getAttribute("content"),
  );
  const res = await page.request.post(`${ADMIN}/admin/tenants`, {
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Csrf-Token": token ?? "",
    },
    data: `name=${encodeURIComponent(name)}&display_name=${encodeURIComponent(display)}`,
  });
  expect(res.ok, `create tenant ${name}: ${res.status}`).toBeTruthy();
  return new URL(res.url()).pathname;
}

test("suggest endpoint: JSON array shape, prefix match, and empty-query refusal", async ({
  page,
}) => {
  const stamp = Date.now();
  const email = `e2e-suggest-${stamp}@acme.dev`;
  await createUser(page, email, "E2E Suggester");

  const res = await page.request.get(`${ADMIN}/admin/users/suggest?q=e2e-suggest`);
  expect(res.ok(), "suggest should answer 200").toBeTruthy();
  const rows: Array<{ id: string; email: string; display_name: string }> = await res.json();
  expect(Array.isArray(rows)).toBeTruthy();
  expect(rows.some((r) => r.email === email)).toBeTruthy();
  expect(rows[0]).toHaveProperty("id");
  expect(rows[0]).toHaveProperty("display_name");

  // Empty query: a named refusal, not a directory dump.
  const empty = await page.request.get(`${ADMIN}/admin/users/suggest?q=`);
  expect(empty.status(), "blank q must be refused").toBe(400);
  const refused: { error: string } = await empty.json();
  expect(refused.error).toContain("q");
});

test("combobox: type, arrow, Enter picks, submit joins the member", async ({ page }) => {
  const stamp = Date.now();
  const tenantPath = await createTenant(page, `e2e-pick-${stamp}`, "E2E Pick Co");
  const email = `e2e-pick-${stamp}@acme.dev`;
  await createUser(page, email, "E2E Picker");

  await page.goto(`${ADMIN}${tenantPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const input = page.locator("#admin-member-email-input");
  await expect(input).toHaveAttribute("role", "combobox");
  await expect(input).toHaveAttribute("aria-expanded", "false");

  // Type the prefix; the debounced fetch fills the listbox.
  await input.click();
  await input.fill(`e2e-pick-${String(stamp).substring(0, 6)}`);
  const listbox = page.locator("#admin-member-email-input-listbox");
  await expect(listbox).toBeVisible({ timeout: 3000 });
  await expect(input).toHaveAttribute("aria-expanded", "true");
  const option = listbox.locator("[role=option]", { hasText: email });
  await expect(option).toHaveCount(1);
  // a11y: the input names the highlighted option.
  await expect
    .poll(() => input.getAttribute("aria-activedescendant"))
    .toBe(await option.getAttribute("id"));

  // Keyboard pick: ArrowDown moves within one option, Enter fills the field.
  await input.press("ArrowDown");
  await input.press("Enter");
  expect(await input.inputValue()).toBe(email);
  await expect(listbox).toBeHidden();
  await expect(input).toHaveAttribute("aria-expanded", "false");

  // Pick a role and submit the form; the member joins.
  await page.locator("input[type=checkbox][name=roles][value=user]").check();
  await page.getByRole("button", { name: "Add member" }).click();
  await page.waitForTimeout(600);
  await expect(
    page.locator("table tbody tr", { hasText: email }).first(),
  ).toBeVisible();
});
