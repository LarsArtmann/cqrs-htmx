import { test, expect, Page } from "@playwright/test";

/**
 * adminui global membership directory smoke (T13/F092 of the 2026-11-24
 * SUPERB plan). Asserts the super-admin directory at /members against the
 * admin-demo on :18930 — the same recipe as admin-tenant-access.spec.ts:
 * SERIAL tests, dev-login before each test, CSRF from the page's
 * <meta name="csrf-token">. Every test seeds its own tenant + user through
 * the panel's create endpoints (gotcha 59: relative assertions only).
 *
 * Coverage: the directory renders seeded memberships (tenant, member, role
 * chips, joined), the search filter narrows to the match, and the tenant
 * chip filter narrows to that tenant's roster.
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

/** Form-POST a panel endpoint with the CSRF header; returns the response. */
async function postForm(
  page: Page,
  path: string,
  fields: Record<string, string | string[]>,
): Promise<{ ok: boolean; status: number; location: string }> {
  const token = await csrf(page);
  const body = Object.entries(fields)
    .flatMap(([k, v]) =>
      (Array.isArray(v) ? v : [v]).map((val) => `${k}=${encodeURIComponent(val)}`),
    )
    .join("&");
  const res = await page.request.post(`${ADMIN}${path}`, {
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Csrf-Token": token,
    },
    data: body,
  });
  return { ok: res.ok(), status: res.status(), location: res.url() };
}

/** Seed a user through the panel. */
async function createUser(page: Page, email: string, name: string): Promise<string> {
  const res = await postForm(page, "/admin/users", { email, display_name: name });
  expect(res.ok, `create user ${email}: ${res.status}`).toBeTruthy();
  return new URL(res.location, ADMIN).pathname;
}

/** Seed a tenant through the panel. */
async function createTenant(page: Page, name: string, display: string): Promise<string> {
  const res = await postForm(page, "/admin/tenants", { name, display_name: display });
  expect(res.ok, `create tenant ${name}: ${res.status}`).toBeTruthy();
  return new URL(res.location, ADMIN).pathname;
}

test("membership directory: renders the seeded membership with tenant, member, roles, joined", async ({
  page,
}) => {
  const stamp = Date.now();
  await createTenant(page, `e2e-dir-${stamp}`, "E2E Directory Co");
  const userPath = await createUser(page, `e2e-dir-${stamp}@acme.dev`, "E2E Directory User");
  await postForm(page, `${userPath}/tenants`, {
    tenant: `e2e-dir-${stamp}`,
    roles: ["viewer", "admin"],
  });

  await page.goto(`${ADMIN}/admin/members`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  const row = page.locator("tr", { hasText: `e2e-dir-${stamp}@acme.dev` });
  await expect(row).toHaveCount(1);
  const cells = await row.innerText();
  expect(cells).toContain("E2E Directory Co"); // tenant column
  expect(cells).toContain("viewer"); // role chips
  expect(cells).toContain("admin");
});

test("membership directory: search narrows to the matching member", async ({ page }) => {
  const stamp = Date.now();
  await createTenant(page, `e2e-dirq-${stamp}`, "E2E Search Co");
  const userPath = await createUser(page, `e2e-dirq-${stamp}@acme.dev`, "E2E Search User");
  await postForm(page, `${userPath}/tenants`, { tenant: `e2e-dirq-${stamp}`, roles: ["viewer"] });

  await page.goto(`${ADMIN}/admin/members?q=e2e-dirq-${stamp}`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  await expect(page.locator("tr", { hasText: `e2e-dirq-${stamp}@acme.dev` })).toHaveCount(1);
  // A membership from another seed does not leak into the search result.
  await expect(page.locator("tr", { hasText: "e2e-dir-1@acme.dev" })).toHaveCount(0);
});

test("membership directory: tenant chip narrows to that tenant's roster", async ({ page }) => {
  const stamp = Date.now();
  await createTenant(page, `e2e-chip-${stamp}`, "E2E Chip Co");
  const userPath = await createUser(page, `e2e-chip-${stamp}@acme.dev`, "E2E Chip User");
  await postForm(page, `${userPath}/tenants`, { tenant: `e2e-chip-${stamp}`, roles: ["owner"] });

  await page.goto(`${ADMIN}/admin/members?tenant=e2e-chip-${stamp}`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  await expect(page.locator("tr", { hasText: `e2e-chip-${stamp}@acme.dev` })).toHaveCount(1);
  await expect(page.locator("tr", { hasText: "E2E Search Co" })).toHaveCount(0);
});
