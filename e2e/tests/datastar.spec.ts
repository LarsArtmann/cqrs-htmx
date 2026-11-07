import { test, expect, type Page } from "@playwright/test";

/**
 * DataStar adapter behavior gate (2026-11-07). Asserts DOM facts against the
 * SHIPPED examples/datastar-demo on :18931 — the artifact that shipped
 * dead-in-browser through every gate (its bootstrap used `data-on:load`,
 * which binds a DOM load event that never fires on ul/div; RC.6 renamed the
 * init hook to `data-init`). These specs exist so that class of client-side
 * regression reds a test run instead of shipping.
 *
 * Coverage: page boot (initial @get patch lands), the create round-trip
 * (data-bind signal -> @post -> elements+signals patches), the live SSE
 * event feed across two tabs, the live-only honesty contract (an event
 * fired while offline stays missed after reconnect; events after reconnect
 * arrive), and a console-error guard (JS-death class).
 *
 * SERIAL: tests share one demo server; count-based assertions depend on
 * execution order.
 */
test.describe.configure({ mode: "serial" });

const DEMO = "http://localhost:18931";

// data-* attribute names carry colons (DataStar RC.6+ syntax); CSS attribute
// selectors must escape them.
const TITLE_INPUT = 'input[data-bind\\:title]';
const TODO_LIST = "#todo-list";
const STATS = "#stats";
const EVENT_LOG = "#event-log .event-entry";

/** Create a todo through the page UI (fills the bound signal, submits). */
async function createTodo(page: Page, title: string) {
  await page.locator(TITLE_INPUT).fill(title);
  await page.getByRole("button", { name: "Add", exact: true }).click();
}

test("boot: initial @get patch populates list and stats (the data-on:load regression)", async ({ page }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });

  // The initial @get('/api/todos') must patch #todo-list — either the empty
  // state or seeded items, never an empty <ul>. This is the exact assertion
  // the shipped demo failed: data-on:load never fired, the list stayed empty.
  await expect
    .poll(async () => page.locator(`${TODO_LIST} > *`).count(), { timeout: 10_000 })
    .toBeGreaterThan(0);

  // The stats patch lands in the same response batch.
  await expect(page.locator(STATS)).toContainText("Total:", { timeout: 10_000 });
});

test("create round-trip: bound signal posts, list+stats+notification patch back", async ({ page }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });
  await expect
    .poll(async () => page.locator(`${TODO_LIST} > *`).count(), { timeout: 10_000 })
    .toBeGreaterThan(0);

  const before = await page.locator(`${TODO_LIST} li.todo-item`).count();

  await createTodo(page, "e2e-roundtrip");

  // The new item renders as a todo row...
  await expect(page.locator(`${TODO_LIST} li.todo-item`)).toHaveCount(before + 1);
  // ...stats reflect it (Total counts every todo)...
  await expect(page.locator(STATS)).toContainText(`Total:`, { timeout: 10_000 });
  // ...the success notification signal shows (data-computed hasNotification).
  await expect(page.locator(".notification")).toContainText("Created: e2e-roundtrip");

  // The bound input signal was cleared by the server's signals patch.
  await expect(page.locator(TITLE_INPUT)).toHaveValue("");
});

test("live feed: an event fired after page load lands in the event log", async ({ page, request }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });

  // Prove the SSE connection is live before firing the event: the feed is
  // live-only (no replay for fresh clients), so the probe event below only
  // appears if the stream is connected.
  const before = await eventCount(page);
  const resp = await request.post(`${DEMO}/api/todos`, {
    headers: { "Content-Type": "application/json" },
    data: { title: "live-probe" },
  });
  expect(resp.ok()).toBeTruthy();

  await expect
    .poll(async () => eventCount(page), { timeout: 15_000 })
    .toBe(before + 1);
});

test("live-only honesty: an event fired while offline stays missed; post-reconnect events arrive", async ({ page, request }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });

  // Connect proof (same as above): one delivered event means the stream is up.
  await request.post(`${DEMO}/api/todos`, {
    headers: { "Content-Type": "application/json" },
    data: { title: "reconnect-proof" },
  });
  await expect
    .poll(async () => eventCount(page), { timeout: 15_000 })
    .toBeGreaterThanOrEqual(1);

  // Sever the page's network. The node-side request fixture is NOT affected
  // by page-context offline, so the server still accepts the command.
  await page.context().setOffline(true);
  const missed = await eventCount(page); // count before the missed event
  await request.post(`${DEMO}/api/todos`, {
    headers: { "Content-Type": "application/json" },
    data: { title: "missed-while-offline" },
  });

  // Reconnect. DataStar retries with backoff (1s * 2 scaler, capped 30s).
  await page.context().setOffline(false);

  // The feed is live-only by design (docs/guides/datastar-integration.md:
  // replay needs the broadcast ring; this demo broadcaster serves live
  // patches) — the missed event must STAY absent. No resurrected facts.
  await page.waitForTimeout(2_500);
  expect(await eventCount(page)).toBe(missed);

  // ...but the connection resumes: a fresh event after reconnect arrives.
  await request.post(`${DEMO}/api/todos`, {
    headers: { "Content-Type": "application/json" },
    data: { title: "after-reconnect" },
  });
  await expect
    .poll(async () => eventCount(page), { timeout: 15_000 })
    .toBe(missed + 1);
  // And the honesty held the whole time: the missed event never appeared.
  expect(await page.locator(EVENT_LOG).count()).toBe(missed + 1);
});

test("multi-tab: a second tab on the same origin receives the same live feed", async ({ page, context }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });
  const tab2 = await context.newPage();
  await tab2.goto(`${DEMO}/`, { waitUntil: "load" });

  // Both tabs boot.
  await expect
    .poll(async () => page.locator(`${TODO_LIST} > *`).count(), { timeout: 10_000 })
    .toBeGreaterThan(0);
  await expect
    .poll(async () => tab2.locator(`${TODO_LIST} > *`).count(), { timeout: 10_000 })
    .toBeGreaterThan(0);

  // A create on tab 1 lands on BOTH tabs' live feeds (fan-out).
  const [before1, before2] = await Promise.all([eventCount(page), eventCount(tab2)]);
  await createTodo(page, "multi-tab-probe");

  await expect
    .poll(async () => eventCount(page), { timeout: 15_000 })
    .toBeGreaterThan(before1);
  await expect
    .poll(async () => eventCount(tab2), { timeout: 15_000 })
    .toBeGreaterThan(before2);
});

test("console guard: no console errors or uncaught page errors during the whole flow", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(`console.error: ${msg.text()}`);
  });
  page.on("pageerror", (err) => errors.push(`pageerror: ${err.message}`));

  await page.goto(`${DEMO}/`, { waitUntil: "load" });
  await expect
    .poll(async () => page.locator(`${TODO_LIST} > *`).count(), { timeout: 10_000 })
    .toBeGreaterThan(0);
  await createTodo(page, "console-guard-probe");
  await expect(page.locator(`${TODO_LIST} li.todo-item`)).toHaveCount(1, { timeout: 10_000 });

  expect(errors, errors.join("\n")).toEqual([]);
});

async function eventCount(page: Page): Promise<number> {
  return page.locator(EVENT_LOG).count();
}
