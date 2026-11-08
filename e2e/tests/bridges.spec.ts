import { test, expect } from "@playwright/test";

/**
 * Bridge modules gate (2026-11-08): asserts the health and auditlog bridge
 * modules' browser surfaces as mounted by the e2e server — the go-health
 * dashboard (cqrshealth.NewDashboard over the projection host) and the
 * samber-do-auditlog live viewer (do lifecycle events over DataStar SSE).
 * POST /api/debug/invoke drives a service invocation through the injector
 * so the audit plugin records it — the deterministic live-event driver.
 */
test.describe.configure({ mode: "serial" });

test("health dashboard renders: title, overall healthy status, live indicator", async ({ page }) => {
  await page.goto("/bridges/health/ui", { waitUntil: "load" });
  await expect(page).toHaveTitle(/Health Dashboard/);
  await expect(page.getByRole("heading", { name: "Health Dashboard" })).toBeVisible();

  // The probe with zero registered projections still reports the overall
  // status — server-rendered, no wait needed (verified against the raw HTML).
  await expect(page.locator("main")).toContainText("healthy", { timeout: 20_000 });
  // The dashboard's own live-connection indicator is present from first paint.
  await expect(page.locator("#conn-state-live")).toBeAttached();
});

test("auditlog viewer renders and its SSE stream connects", async ({ page }) => {
  await page.goto("/bridges/auditlog/", { waitUntil: "load" });
  await expect(page).toHaveTitle(/samber-do-auditlog Live/);

  // The connection badge is DataStar-bound ($connStatus) — reaching
  // "connected" proves the /api/events SSE stream is live in the browser.
  await expect(page.locator("#connection-status")).toHaveText(/connected/, { timeout: 20_000 });
});

test("auditlog viewer streams injector events: invoke grows the events table live", async ({ page, request }) => {
  await page.goto("/bridges/auditlog/", { waitUntil: "load" });
  await expect(page.locator("#connection-status")).toHaveText(/connected/, { timeout: 20_000 });

  // The Events tab lists recorded lifecycle events (startup already
  // registered the probe service, so the table starts non-empty).
  await page.locator("#tab-btn-events").click();
  const rows = page.locator("#tab-events tbody tr:not(.empty-state)");
  const before = await rows.count();
  expect(before).toBeGreaterThan(0);

  // Invoking the do-registered probe service makes the audit plugin record
  // the invocation; the SSE patch must re-render the table in the browser.
  const resp = await request.post("/api/debug/invoke");
  expect(resp.status()).toBe(202);

  await expect(rows).toHaveCount(before + 1, { timeout: 20_000 });
});

test("auditlog services tab shows the do-registered probe service", async ({ page }) => {
  await page.goto("/bridges/auditlog/", { waitUntil: "load" });
  await page.locator("#tab-btn-services").click();
  const services = page.locator("#tab-services tbody tr:not(.empty-state)");
  await expect(services.first()).toBeVisible({ timeout: 20_000 });
  await expect(services.first()).toContainText("bridgeProbeService");
});

test("console guard: no console or page errors across both bridge surfaces", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(`console.error: ${msg.text()}`);
  });
  page.on("pageerror", (err) => errors.push(`pageerror: ${err.message}`));

  await page.goto("/bridges/health/ui", { waitUntil: "load" });
  await expect(page.getByRole("heading", { name: "Health Dashboard" })).toBeVisible();
  await page.goto("/bridges/auditlog/", { waitUntil: "load" });
  await expect(page.locator("#connection-status")).toHaveText(/connected/, { timeout: 20_000 });

  expect(errors, errors.join("\n")).toEqual([]);
});
