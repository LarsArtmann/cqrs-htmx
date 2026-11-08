import { test, expect } from "@playwright/test";

/**
 * Setup bundle gate (2026-11-07): asserts the one-call composition root's
 * mounted reality against the SHIPPED examples/setup-demo on :18933 —
 * session gates on the privileged panels (401 without a session — the
 * enrich-only session middleware never blocks, the explicit gates do),
 * the /health readiness endpoint, the dev-login round-trip (cookie ->
 * 303 -> panel), both panels answering under a session, and the
 * dual-transport broadcast hub (POST /broadcast -> DataStar signal patch
 * lands on /ds-demo — ADR-0050's one-hub-two-transports contract).
 */
test.describe.configure({ mode: "serial" });

const DEMO = "http://localhost:18933";

test("session gates: privileged panels refuse anonymous visitors with 401", async ({ request }) => {
  const admin = await request.get(`${DEMO}/admin/`);
  expect(admin.status()).toBe(401);
  const dash = await request.get(`${DEMO}/dashboard/`);
  expect(dash.status()).toBe(401);
});

test("readiness: /health answers 200 with projections and sse-hub ok", async ({ request }) => {
  const resp = await request.get(`${DEMO}/health`);
  expect(resp.ok()).toBeTruthy();
  const body = await resp.json();
  expect(body.status).toBe("ok");
  expect(body.checks.projections.status).toBe("ok");
  expect(body.checks["sse-hub"].status).toBe("ok");
});

test("dev-login round-trip: cookie issued, redirect lands on the admin panel", async ({ page }) => {
  // The login page at / renders (title proves the bundle's loginpage mount).
  await page.goto(`${DEMO}/`, { waitUntil: "load" });
  await expect(page).toHaveTitle(/Setup Demo/);

  // dev-login sets the session cookie and 303s to /admin/ — the page
  // context carries the cookie, so the final panel renders signed in.
  await page.goto(`${DEMO}/dev-login`, { waitUntil: "load" });
  await expect(page).toHaveURL(/\/admin\/?$/);
  await expect(page).toHaveTitle(/Setup Demo/);
});

test("panels answer under a session: admin and CQRS dashboard both render", async ({ page }) => {
  await page.goto(`${DEMO}/dev-login`, { waitUntil: "load" });

  await page.goto(`${DEMO}/dashboard/`, { waitUntil: "load" });
  await expect(page).toHaveTitle(/CQRS Dashboard/);

  await page.goto(`${DEMO}/admin/`, { waitUntil: "load" });
  await expect(page).toHaveTitle(/Setup Demo/);
});

test("dual-transport hub: /broadcast patches the DataStar signal on /ds-demo", async ({ page, request }) => {
  // The feed (/ds/events) is session-gated by design — event metadata is
  // not public data — so establish the demo session before opening the page.
  await page.goto(`${DEMO}/dev-login`, { waitUntil: "load" });
  await page.goto(`${DEMO}/ds-demo`, { waitUntil: "load" });
  const counter = page.locator("[data-text='$broadcasts']");

  // Rendered baseline (the SDK fills the span from the initial signal).
  await expect(counter).toHaveText(/^\d+$/, { timeout: 10_000 });

  // The server-side counter is GLOBAL and one webServer backs BOTH playwright
  // projects — an absolute 0->2 assertion races the other project's posts.
  // That race WAS the 2026-11-08 "firefox divergence" (firefox's posts landed
  // at 3/4 after chromium ran first; the client was probe-proven faithful —
  // it renders exactly the patch values the server sends). Prove stream
  // liveness with a probe post against the current rendered value, then
  // assert the DELTA for the real pair.
  let live = false;
  let baseline = 0;
  for (let attempt = 0; attempt < 3 && !live; attempt++) {
    baseline = Number(await counter.textContent());
    const resp = await request.post(`${DEMO}/broadcast`);
    expect(resp.status()).toBe(202);
    try {
      await expect(counter).toHaveText(String(baseline + 1), { timeout: 4_000 });
      live = true;
    } catch {
      // stream not live yet — retry with a fresh probe
    }
  }
  expect(live, "DataStar stream never went live on /ds-demo").toBe(true);

  baseline = Number(await counter.textContent());
  for (let i = 0; i < 2; i++) {
    const resp = await request.post(`${DEMO}/broadcast`);
    expect(resp.status()).toBe(202);
  }
  await expect(counter).toHaveText(String(baseline + 2), { timeout: 15_000 });
});

test("console guard: no console or page errors across login page, dashboard, and ds-demo", async ({ page, context }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(`console.error: ${msg.text()}`);
  });
  page.on("pageerror", (err) => errors.push(`pageerror: ${err.message}`));

  // Establish the session via the context's request space (cookies are
  // shared with pages) instead of navigating /dev-login -> /admin/: the
  // admin panel under the bundle's nonce CSP emits a KNOWN inline-style
  // console error (the templ-components --tc-sidebar-w class, documented
  // in AGENTS.md) — not this bundle's regression, and admin-demo's own
  // spec owns that surface. The dev-login -> admin round-trip is covered
  // by the test above.
  // (context.request follows redirects, so this lands on /admin/ 200 — the
  // proof the session took is the guarded pages rendering below.)
  const login = await context.request.get(`${DEMO}/dev-login`);
  expect(login.ok()).toBeTruthy();

  await page.goto(`${DEMO}/`, { waitUntil: "load" });
  await expect(page).toHaveTitle(/Setup Demo/);
  await page.goto(`${DEMO}/dashboard/`, { waitUntil: "load" });
  await expect(page).toHaveTitle(/CQRS Dashboard/);
  await page.goto(`${DEMO}/ds-demo`, { waitUntil: "load" });
  await expect(page.locator("[data-text='$broadcasts']")).toHaveText(/0|1|2/, { timeout: 10_000 });

  expect(errors, errors.join("\n")).toEqual([]);
});
