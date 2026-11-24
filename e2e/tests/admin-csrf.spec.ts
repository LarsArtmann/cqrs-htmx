import { test, expect, Page } from "@playwright/test";

/**
 * adminui CSRF browser-level refusal proof (T28/F205 of the 2026-11-24
 * SUPERB plan). The Go-side integration test (adminui/csrf_test.go) proves
 * the middleware chain refuses a token-less POST with 403 against
 * httptest; THIS spec proves the same refusal in a real browser against
 * the admin-demo on :18930 — the recommended setup stack (httputil
 * CSRFMiddleware over nosurf, the request token surfaced as
 * <meta name="csrf-token"> in the panel layout).
 *
 * The red-without-token proof is a PAIRED control, not absence-of-success:
 * the same endpoint and payload shape SUCCEEDS with the page's meta token
 * and is refused 403 WITHOUT it. Both probes carry the browser context's
 * real cookie jar — the masked csrf_token cookie plus the session — so the
 * refusal is the token VERIFICATION failing (cookie value must never pass
 * as the header token — probed 2026-11-08, gotcha 59), not a missing
 * session. The raw refusal is asserted with maxRedirects: 0 so a redirect
 * could never mask the status.
 *
 * Same recipe as admin-tenant-access.spec.ts: SERIAL mode, dev-login
 * beforeEach, uniquely-named self-seeded fixtures only (gotcha 59: one
 * webServer backs every project — no absolute totals, no shared-fixture
 * deletes). No element-bound JS: one-shot page.evaluate fetches only
 * (gotcha 57 — bound listeners die with the pre-swap DOM under hx-boost).
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

test("CSRF refusal: token-less POST is 403 while the same POST with the meta token creates", async ({
  page,
}) => {
  await page.goto(`${ADMIN}/admin/tenants`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const token = await csrf(page);
  const stamp = Date.now();

  // POSITIVE control (green WITH the token): the same endpoint and payload
  // shape creates the tenant (303 to the detail page; page.request follows
  // it, so ok() is the followed response).
  const ctrlName = `e2e-csrf-ctrl-${stamp}`;
  const created = await page.request.post(`${ADMIN}/admin/tenants`, {
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Csrf-Token": token,
    },
    data: `name=${encodeURIComponent(ctrlName)}&display_name=E2E CSRF Control`,
  });
  expect(created.ok(), `control POST with token: ${created.status()}`).toBeTruthy();

  // NEGATIVE (red WITHOUT the token): identical POST, header omitted. The
  // documented refusal (adminui/csrf_test.go step 3 → httputil
  // handleCSRFRejection) is 403 plain text; maxRedirects: 0 keeps a
  // redirect from ever masking the status.
  const refusedName = `e2e-csrf-refused-${stamp}`;
  const refused = await page.request.post(`${ADMIN}/admin/tenants`, {
    maxRedirects: 0,
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    data: `name=${encodeURIComponent(refusedName)}&display_name=E2E CSRF Refused`,
  });
  expect(refused.status(), "token-less POST must be refused with 403").toBe(403);

  // The refusal is ENFORCING: the refused tenant never reached the store
  // (the name is unique per run, so its absence is race-free under gotcha
  // 59), while the control tenant IS on the list (the q filter itself
  // works — the search row also proves the session stayed healthy).
  await page.goto(`${ADMIN}/admin/tenants?q=${encodeURIComponent(refusedName)}`, {
    waitUntil: "load",
  });
  await page.waitForTimeout(400);
  expect(await page.locator("table tbody tr", { hasText: refusedName }).count()).toBe(0);

  await page.goto(`${ADMIN}/admin/tenants?q=${encodeURIComponent(ctrlName)}`, {
    waitUntil: "load",
  });
  await page.waitForTimeout(400);
  await expect(page.locator("table tbody tr", { hasText: ctrlName }).first()).toBeVisible();
});

test("CSRF refusal: the browser's own same-origin fetch is refused without the token and passes with it", async ({
  page,
}) => {
  await page.goto(`${ADMIN}/admin/tenants`, { waitUntil: "load" });
  await page.waitForTimeout(400);
  const token = await csrf(page);
  const stamp = Date.now();

  // The in-page fetch rides the page's REAL cookie jar (masked csrf_token
  // cookie + session) but carries no token header → refused 403.
  const refusedStatus = await page.evaluate(
    async ({ url, name }) => {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: `name=${encodeURIComponent(name)}&display_name=E2E Browser Refused`,
        credentials: "same-origin",
      });
      return res.status;
    },
    { url: `${ADMIN}/admin/tenants`, name: `e2e-csrf-browser-refused-${stamp}` },
  );
  expect(refusedStatus, "browser fetch without token must be refused with 403").toBe(403);

  // Same channel WITH the meta token → passes (the browser follows the 303
  // to the detail page, so <400). The refusal is therefore caused by the
  // missing token, not by the fetch channel or the endpoint.
  const okStatus = await page.evaluate(
    async ({ url, name, token }) => {
      const res = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Csrf-Token": token,
        },
        body: `name=${encodeURIComponent(name)}&display_name=E2E Browser Control`,
        credentials: "same-origin",
      });
      return res.status;
    },
    { url: `${ADMIN}/admin/tenants`, name: `e2e-csrf-browser-ctrl-${stamp}`, token },
  );
  expect(okStatus, "browser fetch with the meta token must pass").toBeLessThan(400);
});
