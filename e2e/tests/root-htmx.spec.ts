import { test, expect } from "@playwright/test";

/**
 * Root HTMX helper gate (2026-11-07): asserts the partial-render contracts of
 * the SHIPPED examples/basic on :18932 — RenderPartialOrFull (one endpoint,
 * two shapes keyed on HX-Request), OOBHTML (out-of-band badge refresh riding
 * the partial), RenderIf (custom predicate on HX-Target answers the bare
 * count), and the SSE-triggered refresh loop on / (EventSource ->
 * itemCreated -> htmx.ajax re-render).
 *
 * These are the root package's flagship response helpers; until now zero
 * specs exercised them in a browser.
 *
 * SERIAL: request-level tests and browser tests share one in-memory item
 * store; count assertions depend on execution order.
 */
test.describe.configure({ mode: "serial" });

const DEMO = "http://localhost:18932";

test("RenderPartialOrFull: HX-Request switches fragment+OOB vs full document", async ({ request }) => {
  // Full shape (no HTMX headers): a navigable document.
  const full = await request.get(`${DEMO}/demo/items`);
  expect(full.ok()).toBeTruthy();
  const fullBody = await full.text();
  expect(fullBody).toContain("<title>Partial-render showcase");
  expect(fullBody).toContain("<html");

  // Partial shape (HX-Request: true): the list fragment plus the out-of-band
  // badge — never a document wrapper.
  const partial = await request.get(`${DEMO}/demo/items`, {
    headers: { "HX-Request": "true" },
  });
  expect(partial.ok()).toBeTruthy();
  const partialBody = await partial.text();
  expect(partialBody).not.toContain("<html");
  expect(partialBody).not.toContain("<title");
  expect(partialBody).not.toContain("<!DOCTYPE");
  // The OOBHTML block rides along: an element addressed to #item-count with
  // hx-swap-oob, so one response refreshes list AND badge.
  expect(partialBody).toContain('id="item-count"');
  expect(partialBody).toContain("hx-swap-oob");
  // The fragment itself is the item list (seeded rows or the empty state).
  expect(/<ul>|No items yet/.test(partialBody)).toBeTruthy();
});

test("RenderIf: an HX-Request aimed at #item-count answers the bare count", async ({ request }) => {
  // Predicate match (HX-Target: #item-count): the bare count text, no HTML.
  const bare = await request.get(`${DEMO}/demo/items/count`, {
    headers: { "HX-Request": "true", "HX-Target": "#item-count" },
  });
  expect(bare.ok()).toBeTruthy();
  expect(bare.headers()["content-type"]).toMatch(/text\/html/);
  const bareBody = (await bare.text()).trim();
  expect(bareBody).toMatch(/^\d+ items$/);
  expect(bareBody).not.toContain("<");

  // No match: every other request gets the full showcase page.
  const page = await request.get(`${DEMO}/demo/items/count`, {
    headers: { "HX-Request": "true" },
  });
  expect(await page.text()).toContain("<title>Partial-render showcase");
});

test("OOBHTML: a partial refresh swaps the list AND the badge out-of-band", async ({ page, request }) => {
  await page.goto(`${DEMO}/demo/items`, { waitUntil: "load" });
  // Fresh server state (serial: this is the first creator) — badge at 0.
  await expect(page.locator("#item-count")).toHaveText(/0 items/);

  // Create one item node-side; the page does not know yet. Form-encoded —
  // the endpoint's contract since the browser fix (DecodeForm; JSON bodies
  // silently decode as empty forms via ParseForm).
  const created = await request.post(`${DEMO}/api/items`, {
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    data: "name=oob-probe",
  });
  expect(created.ok()).toBeTruthy();

  // The same request an in-page hx-get would issue. htmx must split the
  // response: the <ul> lands in #item-list, the OOB span is EXTRACTED and
  // swapped into the existing #item-count (never dumped into the target).
  await page.evaluate(() =>
    (window as unknown as { htmx: { ajax: (v: string, u: string, s: object) => void } }).htmx.ajax(
      "GET",
      "/demo/items",
      { target: "#item-list", swap: "innerHTML" },
    ),
  );

  await expect(page.locator("#item-count")).toHaveText("1 items", { timeout: 10_000 });
  await expect(page.locator("#item-list li")).toHaveCount(1);
  await expect(page.locator("#item-list li").first()).toContainText("oob-probe");
  // The OOB element was consumed by the swap machinery, not rendered into
  // the target container.
  expect(await page.locator('#item-list [hx-swap-oob]').count()).toBe(0);
});

test("SSE-triggered refresh: Add on / re-renders #items via the itemCreated loop", async ({ page }) => {
  await page.goto(`${DEMO}/`, { waitUntil: "load" });

  // The hx-trigger="load" initial fill resolves (JSON list render).
  await expect(page.locator("#items")).not.toContainText("Loading...", { timeout: 10_000 });

  // The form posts with hx-swap="none" — the ONLY path the new item reaches
  // the list is the EventSource 'itemCreated' listener calling htmx.ajax.
  const name = `sse-probe-${Date.now()}`;
  await page.locator('form[hx-post="/api/items"] input[name="name"]').fill(name);
  await page.getByRole("button", { name: "Add", exact: true }).click();

  await expect(page.locator("#items")).toContainText(name, { timeout: 15_000 });
});

test("console guard: no console or page errors during create and partial refresh", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(`console.error: ${msg.text()}`);
  });
  page.on("pageerror", (err) => errors.push(`pageerror: ${err.message}`));

  await page.goto(`${DEMO}/`, { waitUntil: "load" });
  await expect(page.locator("#items")).not.toContainText("Loading...", { timeout: 10_000 });

  await page.locator('form[hx-post="/api/items"] input[name="name"]').fill("console-guard-probe");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await expect(page.locator("#items")).toContainText("console-guard-probe", { timeout: 15_000 });

  expect(errors, errors.join("\n")).toEqual([]);
});
