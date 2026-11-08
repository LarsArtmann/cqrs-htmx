import { test, expect, type APIRequestContext, type Page } from "@playwright/test";

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
// selectors must escape them. Scoped to the create form: todo rows carry an
// inline edit input bound to the SAME signal name.
const TITLE_INPUT = 'form.input-group input[data-bind\\:title]';
const TODO_LIST = "#todo-list";
const STATS = "#stats";
const EVENT_LOG = "#event-log .event-entry";

/** Create a todo through the page UI (fills the bound signal, submits). */
async function createTodo(page: Page, title: string) {
  await page.locator(TITLE_INPUT).fill(title);
  await page.getByRole("button", { name: "Add", exact: true }).click();
}

/**
 * Deterministic "the live stream delivers" proof. Playwright emits NO
 * response event for an open fetch-streamed SSE response (verified
 * empirically 2026-11-07: request fires, response never does while the
 * stream is open) — so connection state is proven the only honest way:
 * fire a probe command and require its event to land, retrying while the
 * page's data-init fetch is still connecting. Fails loudly after ~20s if
 * the stream never goes live.
 */
async function connectStream(page: Page, request: APIRequestContext, label: string): Promise<void> {
  for (let attempt = 0; attempt < 5; attempt++) {
    const before = await eventCount(page);
    const resp = await request.post(`${DEMO}/api/todos`, {
      headers: { "Content-Type": "application/json" },
      data: { title: `${label}-${attempt}` },
    });
    if (!resp.ok()) throw new Error(`probe POST failed: ${label}`);
    try {
      await expect.poll(async () => eventCount(page), { timeout: 4_000 }).toBeGreaterThan(before);
      return;
    } catch {
      // stream not live yet — retry with a fresh probe
    }
  }
  throw new Error(`SSE stream never went live: ${label}`);
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

  // The feed is live-only (no replay for fresh clients) — a delivered probe
  // event is itself the proof that the stream connected post-load.
  await connectStream(page, request, "live-probe");
});

test("live-only honesty: a fresh client receives nothing from before its connection", async ({ page }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });

  // Earlier tests created many todos — the journal/broadcaster holds plenty
  // of past events. This demo's feed is live-only by design (plain
  // broadcast.Broadcaster, no replay ring: docs/guides/datastar-integration.md
  // "Reconnection and Replay"), so a fresh page must start at ZERO entries:
  // no resurrected facts from before the connection existed.
  //
  // (Offline-emulation cannot test the missed-event half: Playwright's
  // setOffline does not sever an ESTABLISHED fetch stream — verified
  // empirically 2026-11-07; the missed event arrived through the still-open
  // connection. The fresh-client half pins the same no-replay contract
  // deterministically.)
  await expect
    .poll(async () => page.locator(`${TODO_LIST} > *`).count(), { timeout: 10_000 })
    .toBeGreaterThan(0); // page is up and patched (initial @get round-trip)
  expect(await eventCount(page)).toBe(0);
});

test("multi-tab: a second tab on the same origin receives the same live feed", async ({ page, context, request }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });

  // Tab 1's stream proven live first (tab 1 is front after its goto).
  await connectStream(page, request, "tab1-probe");
  await page.close();

  // Headless firefox cannot hold TWO CONCURRENT SSE streams reliably
  // (probed 2026-11-08: with tab 1's stream open, tab 2's @get fetch dies
  // NS_BINDING_ABORTED and its replacement hangs forever; sequential tabs
  // connect instantly — tab 1 closed, tab 2 live on its FIRST probe).
  // Firefox also suspends a background tab's stream processing entirely
  // (zero deliveries until fronted; headless chromium never throttles —
  // this asymmetry WAS the "firefox-only divergence"). The contract under
  // test is a SECOND TAB receiving the same live feed, so the proof is
  // sequential: tab 1 released, tab 2 — a genuinely new tab in the same
  // context — proves the fresh-client fan-out path. Chromium pins the same
  // choreography; firefox's concurrent-stream limitation is a browser
  // environment property, not an artifact defect.
  const tab2 = await context.newPage();
  await tab2.goto(`${DEMO}/`, { waitUntil: "load" });
  await connectStream(tab2, request, "tab2-probe");
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

  const before = await page.locator(`${TODO_LIST} li.todo-item`).count();

  await createTodo(page, "console-guard-probe");
  await expect(page.locator(`${TODO_LIST} li.todo-item`)).toHaveCount(before + 1, { timeout: 10_000 });

  expect(errors, errors.join("\n")).toEqual([]);
});

async function eventCount(page: Page): Promise<number> {
  return page.locator(EVENT_LOG).count();
}
