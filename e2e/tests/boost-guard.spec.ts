import { test, expect } from "@playwright/test";

/**
 * Boost-survival guard (plan 2026-11-08 M7, generalizing bug #5): the
 * dashboard layout boosts navigation (data-hx-boost + wrapper-level
 * hx-target/hx-select on #main-content), and htmx resolves those
 * attributes through the ANCESTOR chain for every ajax element. Any
 * self-fetching element inside the boosted wrapper that does not pin its
 * own attrs selects #main-content from a partial that has none and blanks
 * the page on its first refresh. This spec pins the two known victim
 * classes' defenses and proves they SURVIVE a live swap driven by a real
 * event (/api/debug/emit -> SSE -> boosted refresh).
 */
test.describe.configure({ mode: "serial" });

test("overview PolledRegion pins its own target/select and survives a live event", async ({ page, request }) => {
  await page.goto("/dashboard/", { waitUntil: "load" });

  const region = page.locator("#projection-health");
  await expect(region).toBeVisible();
  // The pinned attrs (htmx.PolledRegion docs: hx-target="this" pins the
  // swap; hx-select="unset" neutralizes inheritance).
  await expect(region).toHaveAttribute("hx-target", "this");
  await expect(region).toHaveAttribute("hx-select", "unset");
  await expect(region).toHaveAttribute(/hx-trigger/, /every 10s, refresh/);

  // Drive a REAL event through the journal: the SSE feed fires the
  // dashboard's refresh triggers, which swap boosted/partial content.
  const before = await page.locator("[data-hx-boost], .app-layout").count();
  const resp = await request.post("/api/debug/emit", { form: { name: "boost-guard-probe" } });
  expect(resp.status()).toBe(202);

  // The events counter on the overview must move (the emit reached the
  // live feed and the page re-rendered boosted content).
  await expect
    .poll(() => page.locator("#stat-total-events").textContent(), { timeout: 20_000 })
    .toContain(/\d+/);

  // After the live re-render: the boost wrapper still exists (the page did
  // not blank) and the PolledRegion STILL carries its pins — the
  // partial re-rendered the region itself (outerHTML contract).
  await expect(page.locator("[data-hx-boost], .app-layout").first()).toBeAttached();
  const after = await page.locator("[data-hx-boost], .app-layout").count();
  expect(after).toBeGreaterThanOrEqual(before);

  await expect(page.locator("#projection-health")).toHaveAttribute("hx-target", "this");
  await expect(page.locator("#projection-health")).toHaveAttribute("hx-select", "unset");
});

test("events filter form pins its own target/select/swap and survives a live event", async ({ page, request }) => {
  await page.goto("/dashboard/events", { waitUntil: "load" });

  const form = page.locator("form.filter-bar");
  await expect(form).toBeVisible();
  await expect(form).toHaveAttribute("hx-get", /\/dashboard\/events$/);
  await expect(form).toHaveAttribute("hx-target", "#main-content");
  await expect(form).toHaveAttribute("hx-select", "#main-content");

  const resp = await request.post("/api/debug/emit", { form: { name: "boost-guard-events" } });
  expect(resp.status()).toBe(202);

  // The events table grows live (dashboard-live.spec owns the deep
  // assertions; here the growth proves the swap actually happened).
  const rows = page.locator("table tbody tr");
  const before = await rows.count();
  await expect
    .poll(async () => page.locator("table tbody tr").count(), { timeout: 20_000 })
    .toBeGreaterThan(before);

  // Post-swap: the form's own pins are intact (a re-rendered form without
  // them would inherit the boost wrapper's target on its NEXT submit and
  // blank the page — the original bug's shape).
  await expect(page.locator("form.filter-bar")).toHaveAttribute("hx-target", "#main-content");
  await expect(page.locator("form.filter-bar")).toHaveAttribute("hx-select", "#main-content");
});
