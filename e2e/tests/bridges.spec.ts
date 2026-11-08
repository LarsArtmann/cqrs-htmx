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

test("health dashboard renders live: page, operational status, SSE patch applied", async ({ page }) => {
  // In-page marker for the first DataStar patch: the definitive proof the
  // /sse feed is wired in the browser (a page-only mount never gets one).
  await page.addInitScript(
    () =>
      new Promise<void>((resolve) => {
        document.addEventListener("datastar-fetch", (e: Event) => {
          const type = (e as CustomEvent).detail?.type;
          if (typeof type === "string" && type.startsWith("datastar-patch-")) {
            (window as any).__sawPatch = true;
          }
        });
        resolve();
      }),
  );

  await page.goto("/bridges/health/ui", { waitUntil: "load" });
  await expect(page).toHaveTitle(/Health Dashboard/);
  await expect(page.getByRole("heading", { name: "Health Dashboard" })).toBeVisible();

  // The probe with zero registered projections reports the overall pass
  // verdict (upstream wording: "All Systems Operational").
  await expect(page.locator("#health-region, main").first()).toContainText("All Systems Operational", { timeout: 20_000 });

  // The SSE feed delivered at least one DataStar patch into the page.
  await expect
    .poll(() => page.evaluate(() => (window as any).__sawPatch === true), { timeout: 20_000 })
    .toBeTruthy();

  // The connection pill's JS state machine reached "live" (live un-hidden,
  // reconnecting/offline hidden). Asserted via the hidden ATTRIBUTE, not
  // visibility: upstream go-health-dashboard v0.13.2 renders all three
  // pills with Tailwind display classes (inline-flex), which defeat the UA
  // [hidden] stylesheet rule — every pill is visually shown at once
  // (verified empirically 2026-11-08; upstream filing candidate).
  await expect
    .poll(
      async () =>
        page.evaluate(() => ({
          live: !document.getElementById("conn-state-live")!.hidden,
          reconnecting: document.getElementById("conn-state-reconnecting")!.hidden,
          offline: document.getElementById("conn-state-offline")!.hidden,
        })),
      { timeout: 20_000 },
    )
    .toEqual({ live: true, reconnecting: true, offline: true });
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
  // One invoke records TWO events (phase before + after rows), so the
  // contract is growth, not an exact row delta.
  const resp = await request.post("/api/debug/invoke");
  expect(resp.status()).toBe(202);

  await expect(async () => {
    expect(await rows.count()).toBeGreaterThan(before);
  }).toPass({ timeout: 20_000 });
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
    if (msg.type() !== "error") return;
    // Firefox's internal favicon loader (resource:///modules/FaviconLoader.sys.mjs)
    // logs a console error when a page's CSP has no img-src and the browser's
    // automatic /favicon.ico request is blocked — the auditlog viewer's
    // upstream CSP is deliberately strict (default-src 'none', samber-do-auditlog
    // v0.12.2). Browser noise, not an artifact regression (chromium stays
    // silent for the same blocked request).
    if (msg.text().includes("FaviconLoader.sys.mjs") && msg.text().includes("favicon.ico")) return;
    errors.push(`console.error: ${msg.text()}`);
  });
  page.on("pageerror", (err) => errors.push(`pageerror: ${err.message}`));

  await page.goto("/bridges/health/ui", { waitUntil: "load" });
  await expect(page.getByRole("heading", { name: "Health Dashboard" })).toBeVisible();
  await expect(page.locator("#health-region, main").first()).toContainText("All Systems Operational", { timeout: 20_000 });
  await page.goto("/bridges/auditlog/", { waitUntil: "load" });
  await expect(page.locator("#connection-status")).toHaveText(/connected/, { timeout: 20_000 });

  expect(errors, errors.join("\n")).toEqual([]);
});
