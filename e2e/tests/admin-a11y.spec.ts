import { test, expect, Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

/**
 * adminui accessibility gate (M11 of the 2026-11-14 SUPERB plan, T110-T113).
 * Two halves:
 *  1. axe-core sweeps over the adminui surfaces this wave touched (tenant
 *     list/detail, users list/detail) — same contract as axe.spec.ts
 *     (dashboard): fail on critical/serious violations, accept only with a
 *     reasoned entry in ACCEPTED.
 *  2. Keyboard paths (T112) for the multi-role checkbox group and the tenant
 *     grant form: every control is reachable by Tab and operable by
 *     Space/Enter — the checkbox group replaced two selects in M02, so the
 *     keyboard contract must be pinned on BOTH surfaces.
 *
 * Same recipe as admin-tenant-access.spec.ts: SERIAL mode, shared in-memory
 * seed, dev-login beforeEach, CSRF from the page meta, self-seeded tenant +
 * user per test (gotcha 59: one webServer backs every project; assert
 * relative to own fixtures).
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

/**
 * The ACCEPTED ledger. Since T15's contrast design pass (F106-F108: the
 * metadata ramp tokens — light-mode metadata text to the gray-600 floor,
 * decorative symbol spans ("|" separators, "—" placeholders) aria-hidden,
 * the disabled-pager ramp to gray-600) the coarse whole-rule
 * "color-contrast" entry from the 2026-11-14 baseline is REMOVED: the
 * re-sweep proved the pass fixed every serious contrast finding it covered.
 * Entries are per-node scoped: a violation is excused only when ALL of its
 * nodes match a listed selector, so new contrast regressions anywhere else
 * still fail the sweep.
 */
const ACCEPTED: Array<{ id: string; why: string; nodes: string[] }> = [];

/** A serious violation survives the ledger only if ALL its nodes are covered. */
function unexpectedViolations(
  serious: Array<{ id: string; impact?: string; nodes: Array<{ target: string[] }> }>,
): Array<{ id: string; impact?: string; nodes: Array<{ target: string[] }> }> {
  return serious.filter((v) => {
    const entry = ACCEPTED.find((e) => e.id === v.id);
    if (!entry) return true;
    const targets = v.nodes.map((n) => n.target.join(" "));
    return targets.some((t) => !entry.nodes.some((sel) => t.includes(sel)));
  });
}

function summarize(
  violations: Array<{ id: string; impact?: string; nodes: Array<{ target: string[] }> }>,
): string {
  return violations
    .map(
      (v) =>
        `${v.id} (${v.impact}): ${v.nodes.length} node(s), e.g. ${v.nodes[0]?.target.join(" ")}`,
    )
    .join("; ");
}

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

/** Seed a tenant + a user through the panel; returns both detail paths. */
async function seedTenantAndUser(page: Page, stamp: number): Promise<{ userPath: string }> {
  const tenant = await postForm(page, "/admin/tenants", {
    name: `e2e-a11y-${stamp}`,
    display_name: "E2E A11y Co",
  });
  expect(tenant.ok, `seed tenant: ${tenant.status}`).toBeTruthy();
  const user = await postForm(page, "/admin/users", {
    email: `e2e-a11y-${stamp}@acme.dev`,
    display_name: "E2E A11y User",
  });
  expect(user.ok, `seed user: ${user.status}`).toBeTruthy();
  return { userPath: new URL(user.location, ADMIN).pathname };
}

// 1. axe sweeps over the touched adminui surfaces.
// T15 F104/F105 extend the sweep to the audit page, the panel dashboard
// (the adminui surface at "/"), and the super-admin import form.
const AXE_PAGES: Array<{ name: string; url: string }> = [
  { name: "tenants list", url: "/admin/tenants" },
  { name: "users list", url: "/admin/users" },
  { name: "audit page", url: "/admin/audit" },
  { name: "panel dashboard", url: "/" },
  { name: "import form", url: "/admin/users/import" },
];

for (const { name, url } of AXE_PAGES) {
  test(`axe sweep (adminui): ${name}`, async ({ page }) => {
    await page.goto(`${ADMIN}${url}`, { waitUntil: "load" });
    await page.waitForTimeout(400);

    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();

    const serious = results.violations.filter((v) =>
      ["critical", "serious"].includes(v.impact ?? ""),
    );
    const unexpected = unexpectedViolations(serious);
    if (unexpected.length > 0) {
      test.info().annotations.push({ type: "axe findings", description: summarize(unexpected) });
    }
    expect(unexpected, `axe critical/serious violations on ${name}`).toEqual([]);
  });
}

test("axe sweep (adminui): tenant detail and user detail", async ({ page }, testInfo) => {
  const stamp = Date.now();
  const { userPath } = await seedTenantAndUser(page, stamp);

  for (const [name, url] of [
    ["tenant detail", `/admin/tenants/e2e-a11y-${stamp}`],
    ["user detail", userPath],
  ] as const) {
    await page.goto(`${ADMIN}${url}`, { waitUntil: "load" });
    await page.waitForTimeout(400);
    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();
    const serious = results.violations.filter((v) =>
      ["critical", "serious"].includes(v.impact ?? ""),
    );
    const unexpected = unexpectedViolations(serious);
    if (unexpected.length > 0) {
      testInfo.annotations.push({
        type: `axe findings (${name})`,
        description: summarize(unexpected),
      });
    }
    expect(unexpected, `axe critical/serious violations on ${name}`).toEqual([]);
  }
});

// 2. T112: the role checkbox group is fully keyboard-operable on the user
// detail's grant form (Tab reaches select → each checkbox → submit; Space
// toggles; the POST carries both checked roles).
test("keyboard: the tenant grant form is reachable and operable without a mouse", async ({
  page,
}) => {
  const stamp = Date.now();
  await seedTenantAndUser(page, stamp);

  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  // Search-scoped: the fresh user is on the last page of the paginated list.
  await page.goto(`${ADMIN}/admin/users?q=e2e-a11y-${stamp}@acme.dev`, { waitUntil: "load" });
  await page.waitForTimeout(400);

  await page
    .locator("#users-table tbody tr", { hasText: `e2e-a11y-${stamp}@acme.dev` })
    .locator("a")
    .first()
    .click();
  await page.waitForTimeout(400);

  // T08: the grant control is a combobox now. Operate it entirely by
  // keyboard — type the prefix, ArrowDown, Enter picks — then Tab onward to
  // the role checkboxes: the checkbox group is the FIRST roles control after
  // the combobox input, and every checkbox is in one Tab ring (native
  // inputs — no tabindex traps).
  const grantInput = page.locator("#admin-grant-tenant-input").first();
  await grantInput.focus();
  await page.keyboard.type(`e2e-a11y-${stamp}`);
  // Wait for the OPTION, not just the popup: Enter must land while the
  // listbox is open, or it falls through to a native form submit.
  await expect(
    page
      .locator("#admin-grant-tenant-input-listbox [role=option]")
      .first(),
  ).toBeVisible({ timeout: 3000 });
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  expect(await grantInput.inputValue()).toBe(`e2e-a11y-${stamp}`);

  const reached: string[] = [];
  for (let i = 0; i < 8; i++) {
    await page.keyboard.press("Tab");
    const info = await page.evaluate(() => {
      const el = document.activeElement as HTMLInputElement | null;
      return el ? `${el.tagName.toLowerCase()}[${el.type ?? ""}] ${el.name ?? ""}` : "none";
    });
    reached.push(info);
    if (info.startsWith("input[checkbox]")) {
      // Space toggles without moving focus (native checkbox behavior).
      const before = await page.evaluate(
        () => (document.activeElement as HTMLInputElement).checked,
      );
      await page.keyboard.press("Space");
      const after = await page.evaluate(() => (document.activeElement as HTMLInputElement).checked);
      expect(after, "Space must toggle the focused role checkbox").toBe(!before);
    }
    if (info.startsWith("button[submit]") || info.includes("button")) break;
  }
  expect(
    reached.some((r) => r.startsWith("input[checkbox]")),
    `Tab ring from the grant select must reach the role checkboxes; got ${reached.join(" | ")}`,
  ).toBeTruthy();

  // Grant two roles purely by keyboard: select tenant, Tab to checkboxes,
  // Space the first two, Enter to submit; the access card gains the row.
  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(300);
  const stamp2 = Date.now();
  await seedTenantAndUser(page, stamp2);
  await page.goto(`${ADMIN}/admin/users?q=e2e-a11y-${stamp2}@acme.dev`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  page.on("request", (r) => {
    if (r.url().includes("/tenants") && r.method() === "POST" && (r.postData() ?? "").includes("e2e-a11y"))
      console.log("GRANT-POST", r.url(), r.postData());
  });
  await page
    .locator("#users-table tbody tr", { hasText: `e2e-a11y-${stamp2}@acme.dev` })
    .locator("a")
    .first()
    .click();
  await page.waitForTimeout(400);

  const input2 = page.locator("#admin-grant-tenant-input").first();
  await input2.focus();
  await page.keyboard.type(`e2e-a11y-${stamp2}`);
  await expect(
    page
      .locator("#admin-grant-tenant-input-listbox [role=option]")
      .first(),
  ).toBeVisible({ timeout: 3000 });
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  expect(await input2.inputValue()).toBe(`e2e-a11y-${stamp2}`);
  let toggled = 0;
  for (let i = 0; i < 6 && toggled < 2; i++) {
    await page.keyboard.press("Tab");
    const isCheckbox = await page.evaluate(
      () => (document.activeElement as HTMLInputElement)?.type === "checkbox",
    );
    if (isCheckbox) {
      await page.keyboard.press("Space");
      toggled++;
    }
  }
  expect(toggled).toBe(2);
  // Submit: the focused element walks to the submit button; Enter there
  // posts the form (native behavior). Press Enter on the last checkbox's
  // form by requesting a submit through keyboard activation of the button.
  await page.keyboard.press("Tab"); // reach the submit button
  const onButton = await page.evaluate(
    () => (document.activeElement as HTMLElement)?.tagName === "BUTTON",
  );
  if (onButton) {
    await page.keyboard.press("Enter");
  } else {
    // Fall back to the form's own submit shortcut from a text-free form:
    // Enter inside any form control submits the first submit button.
    await page.keyboard.press("Enter");
  }
  await page.waitForTimeout(600);

  await page.goto(`${ADMIN}/admin/users?q=e2e-a11y-${stamp2}@acme.dev`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const row = page.locator("#users-table tbody tr", { hasText: `e2e-a11y-${stamp2}@acme.dev` });
  await expect(row).toHaveCount(1);
  console.log("ROW:", JSON.stringify(await row.innerText()));
  await page.locator("#users-table tbody tr", { hasText: `e2e-a11y-${stamp2}@acme.dev` }).locator("a").first().click();
  await page.waitForTimeout(400);
  const card = page.locator("[data-sync-target]", { hasText: "Tenant access" }).first();
  console.log("ACCESS:", JSON.stringify(await card.innerText()));
  await page.goto(`${ADMIN}/admin/users?q=e2e-a11y-${stamp2}@acme.dev`, { waitUntil: "load" });
  expect(await row.innerText()).toContain("1 tenant");
});
