import { test, expect } from "@playwright/test";

/**
 * Session posture gate (2026-11-08, plan M11): pins the identity endpoints'
 * auth contract on the e2e server's loginpage surface — GET /auth/me is
 * session-gated (401 JSON anonymous, 200 with the user under a session),
 * POST /auth/register issues a browser-usable session cookie, and POST
 * /auth/logout revokes it everywhere. Uses the page's request space (cookies
 * shared with the browser context) so the cookie path is proven in the
 * browser, not just over curl.
 */
test.describe.configure({ mode: "serial" });

const email = () => `m11-${Date.now()}@example.com`;

test("posture: /auth/me refuses anonymous callers with 401 JSON", async ({ request }) => {
  const resp = await request.get("/auth/me");
  expect(resp.status()).toBe(401);
  expect(resp.headers()["content-type"]).toContain("application/json");
  await expect(resp.json()).resolves.toMatchObject({ error: expect.any(String) });
});

test("register issues a browser session: /auth/me answers 200 with the user", async ({ page, context }) => {
  const address = email();

  // The CSRF token lives in the /login page's meta tag; the request space
  // shares cookies with the page, so fetching it through the page context
  // keeps the csrf_token cookie paired with the masked token.
  await page.goto("/login", { waitUntil: "load" });
  const csrf = await page.locator('meta[name="csrf-token"]').getAttribute("content");
  expect(csrf).toBeTruthy();

  const reg = await context.request.post("/auth/register", {
    headers: { "X-CSRF-Token": csrf ?? "" },
    data: { email: address, display_name: "M11 Posture" },
  });
  expect(reg.status()).toBe(201);

  // The session cookie rides the shared context: /auth/me answers 200.
  const me = await context.request.get("/auth/me");
  expect(me.status()).toBe(200);
  const user = await me.json();
  expect(user.email).toBe(address);
  expect(user.id).toBeTruthy();

  // And the browser itself carries the session: the gated landing page
  // renders the signed-in email from the REAL session.
  await page.goto("/post-login", { waitUntil: "load" });
  await expect(page.locator("#post-login-email")).toHaveText(address);
});

test("logout revokes the session: /auth/me 401 again, gated page bounces", async ({ page, context }) => {
  // Playwright gives every test a fresh context — establish this test's own
  // session before revoking it (register issues the session cookie).
  await page.goto("/login", { waitUntil: "load" });
  const csrf = await page.locator('meta[name="csrf-token"]').getAttribute("content");
  const address = email();
  const reg = await context.request.post("/auth/register", {
    headers: { "X-CSRF-Token": csrf ?? "" },
    data: { email: address, display_name: "M11 Logout" },
  });
  expect(reg.status()).toBe(201);
  expect((await context.request.get("/auth/me")).status()).toBe(200);

  const out = await context.request.post("/auth/logout", {
    headers: { "X-CSRF-Token": csrf ?? "" },
  });
  expect(out.status()).toBe(200);
  await expect(out.json()).resolves.toMatchObject({ status: "logged_out" });

  const me = await context.request.get("/auth/me");
  expect(me.status()).toBe(401);

  // The session-gated landing page bounces anonymous visitors to /login.
  await page.goto("/post-login", { waitUntil: "load" });
  await expect(page).toHaveURL(/\/login$/);
});
