import { test, expect } from "@playwright/test";

/**
 * Dashboard live/replay depth gate (2026-11-07): drives the dashboard's SSE
 * contract deterministically through POST /api/debug/emit (one event lands in
 * the journal AND on the bus) — the live append (connected page gains the row
 * without reload), the journal replay honesty (a fresh page load still lists
 * the event — the deliberate counterpart to the datastar demo's live-only
 * feed, which serves zero past events to fresh clients), and the time-travel
 * slider (drag updates the display; release navigates to the chosen version).
 *
 * SERIAL: the journal is shared server state; row counts build on each other.
 */
test.describe.configure({ mode: "serial" });

const DEBUG_TYPE = "E2e.debugEmitted";

test("live append: an emitted event lands a row at the top of the events table", async ({ page, request }) => {
  await page.goto("/dashboard/events");
  await expect(page.locator("#events-tbody tr").first()).toBeVisible();
  const before = await page.locator("#events-tbody tr").count();

  const name = `live-probe-${Date.now()}`;
  const resp = await request.post("/api/debug/emit", { form: { name } });
  expect(resp.status()).toBe(202);

  // No reload: the SSE feed appends the row client-side (class new-row,
  // inserted at the top of #events-tbody).
  await expect
    .poll(async () => page.locator("#events-tbody tr").count(), { timeout: 15_000 })
    .toBe(before + 1);
  await expect(page.locator("#events-tbody tr").first()).toContainText(DEBUG_TYPE);
  await expect(page.locator("#events-tbody tr").first()).toContainText("E2e");
});

test("journal replay: a fresh page load still lists the emitted event", async ({ page }) => {
  // The counterpoint to the datastar demo's live-only feed: the dashboard
  // journal persists, so a cold client sees past events on first render.
  await page.goto("/dashboard/events");
  await expect(page.locator("#events-tbody tr").first()).toBeVisible();
  await expect(page.locator("#events-tbody")).toContainText(DEBUG_TYPE);
});

test("multi-tab live feed: a second connected dashboard gains the row too", async ({ page, context, request }) => {
  await page.goto("/dashboard/events");
  const tab2 = await context.newPage();
  await tab2.goto("/dashboard/events");
  await expect(page.locator("#events-tbody tr").first()).toBeVisible();
  await expect(tab2.locator("#events-tbody tr").first()).toBeVisible();

  const [before1, before2] = await Promise.all([
    page.locator("#events-tbody tr").count(),
    tab2.locator("#events-tbody tr").count(),
  ]);

  const resp = await request.post("/api/debug/emit", { form: { name: "multi-tab-probe" } });
  expect(resp.status()).toBe(202);

  await expect
    .poll(async () => page.locator("#events-tbody tr").count(), { timeout: 15_000 })
    .toBe(before1 + 1);
  await expect
    .poll(async () => tab2.locator("#events-tbody tr").count(), { timeout: 15_000 })
    .toBe(before2 + 1);
});

test("time travel: the slider updates the display while dragging and navigates on release", async ({ page }) => {
  await page.goto("/dashboard/time-travel");

  // A seeded User stream has exactly 2 versions.
  const userRow = page.locator("table tbody tr", { hasText: "User" }).first();
  await expect(userRow).toBeVisible();
  await userRow.getByRole("link").first().click();
  await expect(page.locator(".page-subtitle", { hasText: "Viewing version 2 of 2" })).toBeVisible();

  const slider = page.locator("#version-slider");
  await expect(slider).toHaveValue("2", { timeout: 10_000 });

  // The delegated input wiring updates the live display WITHOUT navigating
  // (drag feedback). Proven with a bubbling input event — the wiring is
  // document-delegated, so this also fails if anyone re-binds it to the
  // element and breaks boosted navigation again.
  await page.evaluate(() => {
    const s = document.getElementById("version-slider");
    if (!s) throw new Error("slider missing");
    s.value = "1";
    s.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await expect(page.locator("#slider-version-display")).toHaveText("1");

  // Arrow key (the page's documented interaction): the keydown handler sets
  // the value and dispatches change -> delegated listener navigates to ?v=1
  // (the server-rendered state). Pinned end-to-end. (Reset to v2 first: the
  // probe above left the slider at 1, and ArrowLeft from 1 clamps at min.)
  await page.evaluate(() => {
    const s = document.getElementById("version-slider");
    if (s) s.value = "2";
  });
  await slider.focus();
  await page.keyboard.press("ArrowLeft");
  await expect(page).toHaveURL(/\/time-travel\/User\/[^\s]+\?v=1$/);
  await expect(page.locator(".page-subtitle", { hasText: "Viewing version 1 of 2" })).toBeVisible();
  await expect(page.locator("#version-slider")).toHaveValue("1");
});

test("console guard: no console or page errors during live feed and time travel", async ({ page, request }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(`console.error: ${msg.text()}`);
  });
  page.on("pageerror", (err) => errors.push(`pageerror: ${err.message}`));

  await page.goto("/dashboard/events");
  await expect(page.locator("#events-tbody tr").first()).toBeVisible();
  const resp = await request.post("/api/debug/emit", { form: { name: "console-guard-probe" } });
  expect(resp.status()).toBe(202);
  await expect(page.locator("#events-tbody tr").first()).toContainText(DEBUG_TYPE, { timeout: 15_000 });

  await page.goto("/dashboard/time-travel");
  await expect(page.locator("table tbody tr").first()).toBeVisible();

  expect(errors, errors.join("\n")).toEqual([]);
});
