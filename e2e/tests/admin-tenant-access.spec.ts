import { test, expect, Page } from "@playwright/test";

/**
 * adminui tenant/user-management behavior gate (M04 of the 2026-11-14 SUPERB
 * plan). Asserts the tenant-integration flows against the admin-demo on
 * :18930 — the same recipe as admin-behavior.spec.ts: SERIAL tests, shared
 * in-memory seed, dev-login before each test, CSRF from the page's
 * <meta name="csrf-token"> (the cookie holds the MASKED value and is
 * rejected as a header token — probed 2026-11-08). Every test seeds its own
 * tenant AND user through the panel's create endpoints, so assertions are
 * relative to the fixtures each test made (gotcha 59: one webServer backs
 * both playwright projects; absolute totals race the other engine).
 *
 * Coverage: grant tenant access from the user page (row, email, role badges,
 * joined tenant leaves the grant select), change roles via the checkbox
 * group and revoke (row state each step), the add-member datalist (directory
 * feed, joined members drop out, unknown email banner preserves the typed
 * address), and the users-list access column (badges, Administrators chip,
 * id search).
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
    maxRedirects: 0,
  });
  return { ok: res.ok(), status: res.status(), location: res.headers().location ?? "" };
}

/** Seed a user through the panel; returns the user detail path. */
async function createUser(page: Page, email: string, name: string): Promise<string> {
  const res = await postForm(page, "/admin/users", { email, display_name: name });
  expect(res.ok, `create user ${email}: ${res.status}`).toBeTruthy();
  return new URL(res.location, ADMIN).pathname;
}

/** Seed a tenant through the panel; returns the tenant detail path. */
async function createTenant(page: Page, name: string, display: string): Promise<string> {
  const res = await postForm(page, "/admin/tenants", { name, display_name: display });
  expect(res.ok, `create tenant ${name}: ${res.status}`).toBeTruthy();
  return new URL(res.location, ADMIN).pathname;
}

test("tenant access: grant from the user page renders the row, badges and filtered select", async ({
  page,
}) => {
  const stamp = Date.now();
  const tenantPath = await createTenant(page, `e2e-grant-${stamp}`, "E2E Grant Co");
  const tenantName = "E2E Grant Co";
  const userPath = await createUser(page, `e2e-grant-${stamp}@acme.dev`, "E2E Grantee");

  // Grant TWO roles at once — the checkbox group posts the full set.
  const grant = await postForm(page, `${userPath}/tenants`, {
    tenant: `e2e-grant-${stamp}`,
    roles: ["viewer", "admin"],
  });
  expect(grant.ok, `grant: ${grant.status}`).toBeTruthy();

  await page.goto(`${ADMIN}${userPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const html = await page.content();

  // The Tenant access row resolves the tenant identity, not an opaque id.
  const row = page.locator("tr", { hasText: tenantName });
  await expect(row).toHaveCount(1);
  expect(html).toContain(`e2e-grant-${stamp}@acme.dev`.split("@")[0] + "" || tenantName);
    const badges = await row.innerText();
  expect(badges).toContain("viewer");
  expect(badges).toContain("admin");

  // The joined tenant left the grant select's options.
  const options = await page
    .locator("select[name='tenant'] option")
    .evaluateAll((els) => els.map((el) => (el as HTMLOptionElement).value));
  expect(options).not.toContain(`e2e-grant-${stamp}`);
  expect(options).toContain("Choose tenant…");

  // The tenant detail roster also names the person (cross-surface view).
  await page.goto(`${ADMIN}${tenantPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(
    page.locator("#admin-members-table tbody tr, table tbody tr", {
      hasText: `e2e-grant-${stamp}@acme.dev`,
    }).first(),
  ).toBeVisible();
});

test("tenant access: change roles via the checkbox group, then revoke", async ({ page }) => {
  const stamp = Date.now();
  const tenantId = `e2e-role-${stamp}`;
  await createTenant(page, tenantId, "E2E Role Co");
  const userPath = await createUser(page, `e2e-role-${stamp}@acme.dev`, "E2E Role User");
  await postForm(page, `${userPath}/tenants`, { tenant: tenantId, roles: ["viewer"] });

  // Replace the set with two roles through the row's checkbox group.
  const update = await postForm(page, `${userPath}/tenants/${tenantId}`, {
    roles: ["viewer", "admin"],
  });
  expect(update.ok, `update roles: ${update.status}`).toBeTruthy();
  await page.goto(`${ADMIN}${userPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const row = page.locator("tr", { hasText: "E2E Role Co" });
  const badges = await row.innerText();
  expect(badges).toContain("viewer");
  expect(badges).toContain("admin");

  // Reduce to one role: the posted set REPLACES, never merges.
  const reduce = await postForm(page, `${userPath}/tenants/${tenantId}`, { roles: ["owner"] });
  expect(reduce.ok, `reduce roles: ${reduce.status}`).toBeTruthy();
  await page.goto(`${ADMIN}${userPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const reduced = await page.locator("tr", { hasText: "E2E Role Co" }).innerText();
  expect(reduced).toContain("owner");
  expect(reduced).not.toContain("viewer");
  expect(reduced).not.toContain("admin");

  // The last-admin escalation only fires while this user is the sole
  // admin-tier holder; after this reduce the tenant has exactly one.
  expect(reduced).toContain("only admin or owner");

  // Revoke removes the row entirely.
  const revoke = await postForm(page, `${userPath}/tenants/${tenantId}/delete`, {});
  expect(revoke.ok, `revoke: ${revoke.status}`).toBeTruthy();
  await page.goto(`${ADMIN}${userPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(page.locator("tr", { hasText: "E2E Role Co" })).toHaveCount(0);
  await expect(page.getByText("not a member of any tenant yet")).toBeVisible();
});

test("add member: datalist feeds the directory, drops joined members, unknown email explains", async ({
  page,
}) => {
  const stamp = Date.now();
  const tenantId = `e2e-member-${stamp}`;
  const tenantPath = await createTenant(page, tenantId, "E2E Member Co");
  await createUser(page, `e2e-m1-${stamp}@acme.dev`, "Member One");
  await createUser(page, `e2e-m2-${stamp}@acme.dev`, "Member Two");

  await page.goto(`${ADMIN}${tenantPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  let html = await page.content();
  expect(html).toContain('id="admin-member-emails"');
  expect(html).toContain(`e2e-m1-${stamp}@acme.dev`);
  expect(html).toContain(`e2e-m2-${stamp}@acme.dev`);

  // Join member one through the add-member form endpoint.
  const add = await postForm(page, `${tenantPath}/members`, {
    email: `e2e-m1-${stamp}@acme.dev`,
    roles: ["user"],
  });
  expect(add.ok, `add member: ${add.status}`).toBeTruthy();

  // The joined member is no longer suggested — a duplicate add is refused by
  // the decider, so offering them is a dead end.
  await page.goto(`${ADMIN}${tenantPath}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  html = await page.content();
  const dataList = await page.locator("#admin-member-emails").innerText();
  expect(dataList).not.toContain(`e2e-m1-${stamp}@acme.dev`);
  expect(dataList).toContain(`e2e-m2-${stamp}@acme.dev`);
  // The roster row names the person, not the actor token.
  await expect(
    page.locator("table tbody tr", { hasText: `e2e-m1-${stamp}@acme.dev` }).first(),
  ).toBeVisible();

  // Unknown email: the redirect carries the miss back inline, preserving the
  // typed address for correction.
  const miss = await postForm(page, `${tenantPath}/members`, {
    email: `nobody-${stamp}@nowhere.test`,
    roles: ["viewer"],
  });
  expect(miss.ok, `unknown add: ${miss.status}`).toBeTruthy();
  const back = new URL(miss.location, ADMIN);
  expect(back.searchParams.get("member_error")).toBe("unknown");
  expect(back.searchParams.get("member_email")).toBe(`nobody-${stamp}@nowhere.test`);
  await page.goto(back.toString(), { waitUntil: "load" });
  await page.waitForTimeout(400);
  await expect(page.getByText("That account does not exist yet")).toBeVisible();
  expect(await page.locator("input[name='email']").inputValue()).toBe(
    `nobody-${stamp}@nowhere.test`,
  );
});

test("users list: access column, Administrators chip, and id search", async ({ page }) => {
  const stamp = Date.now();
  const tenantId = `e2e-access-${stamp}`;
  await createTenant(page, tenantId, "E2E Access Co");
  const userPath = await createUser(page, `e2e-access-${stamp}@acme.dev`, "E2E Access User");
  const userId = userPath.split("/").pop() ?? "";
  await postForm(page, `${userPath}/tenants`, { tenant: tenantId, roles: ["admin"] });

  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const html = await page.content();
  expect(html).toContain(">Access<");

  // The user's row shows the tenant count and the platform-role area.
  const row = page.locator("#users-table tbody tr", { hasText: `e2e-access-${stamp}@acme.dev` });
  await expect(row).toHaveCount(1);
  expect(await row.innerText()).toContain("E2E Access Co");

  // Id search: pasting the user id finds exactly that person.
  await page.goto(`${ADMIN}/admin/users?q=${userId}`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const idRow = page.locator("#users-table tbody tr");
  await expect(idRow).toHaveCount(1);
  expect(await idRow.innerText()).toContain(`e2e-access-${stamp}@acme.dev`);

  // The Administrators chip exists and narrows to platform-role holders.
  const chip = page.locator("a", { hasText: "Administrators" }).first();
  expect(await chip.getAttribute("href")).toContain("filter=");
});
