import { test, expect, type APIRequestContext, type Page } from "@playwright/test";

/**
 * Client tracker gate (task T18 of the native query log plan): the REAL
 * embedded tracker.js in a real browser against the REAL TrackHandler with
 * session attribution. The e2e server records every validated signal in a
 * test-only sink that these specs read back through /api/debug/signals.
 *
 * Every test resets the sink first: the e2e battery shares one server
 * across projects, so assertions are made on the signals a test caused.
 */

const E2E = "http://localhost:18923";

/** One record as the sink returns it (the zero fields are omitted). */
type Signal = {
  kind: string;
  path: string;
  referrer?: string;
  name?: string;
  target?: string;
  visibleMs?: number;
  userId?: string;
  clientId?: string;
};

declare global {
  interface Window {
    cqrsTracker: { flush(): void; enabled(): boolean };
  }
}

async function resetSignals(request: APIRequestContext): Promise<void> {
  const reset = await request.post(`${E2E}/api/debug/signals/reset`);
  expect(reset.status()).toBe(204);
}

async function signals(request: APIRequestContext): Promise<Signal[]> {
  const response = await request.get(`${E2E}/api/debug/signals`);
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as Signal[];
}

/** Polls the sink until `pick` finds a signal; returns it. */
async function awaitSignal(
  request: APIRequestContext,
  what: string,
  pick: (all: Signal[]) => Signal | undefined,
): Promise<Signal> {
  await expect
    .poll(async () => pick(await signals(request)), { message: what, timeout: 15_000 })
    .toBeTruthy();
  const found = pick(await signals(request));
  expect(found).toBeTruthy();
  return found as Signal;
}

/** Signs the tracking user in; returns the user key every signal must carry. */
async function signIn(request: APIRequestContext): Promise<string> {
  const login = await request.post(`${E2E}/api/debug/tracking-login`);
  expect(login.ok()).toBeTruthy();
  return ((await login.json()) as { userId: string }).userId;
}

test("page view and dwell arrive with the user and device keys", async ({ page, context }) => {
  // context.request shares the context's cookie jar with the page (the
  // isolated `request` fixture does not), so the login cookie rides beacons.
  const api = context.request;
  await resetSignals(api);
  const userId = await signIn(api);

  await page.goto(`${E2E}/tracking`, { waitUntil: "load" });

  const deviceKey = await page.evaluate(() => window.localStorage.getItem("cqrs-htmx-client-id"));
  expect(deviceKey).toMatch(/^[0-9A-HJKMNP-TV-Z]{26}$/);

  await page.evaluate(() => window.cqrsTracker.flush());

  const view = await awaitSignal(api, "the page view reaches the sink", (all) =>
    all.find((signal) => signal.kind === "pageview"),
  );
  expect(view).toMatchObject({ kind: "pageview", path: "/tracking", userId, clientId: deviceKey });

  // A visible second, then an in-app navigation ends the page view: its
  // dwell is one engagement signal, and the new location one page view whose
  // referrer is the page it came from.
  await page.waitForTimeout(1200);
  await page.evaluate(() => {
    history.pushState({}, "", "/tracking?step=2");
    window.dispatchEvent(new PopStateEvent("popstate"));
    window.cqrsTracker.flush();
  });

  const dwell = await awaitSignal(api, "the dwell reaches the sink", (all) =>
    all.find((signal) => signal.kind === "engagement"),
  );
  expect(dwell.visibleMs).toBeGreaterThanOrEqual(1000);
  expect(dwell).toMatchObject({ path: "/tracking", userId, clientId: deviceKey });

  const next = await awaitSignal(api, "the in-app navigation is its own page view", (all) => {
    const views = all.filter((signal) => signal.kind === "pageview");
    return views.length === 2 ? views[1] : undefined;
  });
  expect(next).toMatchObject({ path: "/tracking", userId, clientId: deviceKey, referrer: `${E2E}/tracking` });
});

test("click and impression arrive named after their elements", async ({ page, context }) => {
  const api = context.request;
  await resetSignals(api);

  await page.goto(`${E2E}/tracking`, { waitUntil: "load" });
  await page.click("#buy");
  await page.evaluate(() => window.cqrsTracker.flush());

  const click = await awaitSignal(api, "the click reaches the sink", (all) =>
    all.find((signal) => signal.kind === "click"),
  );
  expect(click).toMatchObject({ path: "/tracking", name: "buy", target: "button#buy" });
  expect(click.userId).toBeUndefined();

  const impression = await awaitSignal(api, "the impression reaches the sink", (all) =>
    all.find((signal) => signal.kind === "impression"),
  );
  expect(impression).toMatchObject({ path: "/tracking", name: "promo", target: "div#promo" });
  expect(impression.userId).toBeUndefined();
});

/** Clicks the tracked button and asks the tracker to send what it queued. */
async function interactAndFlush(page: Page): Promise<void> {
  await page.click("#buy");
  await page.evaluate(() => window.cqrsTracker.flush());
  // A queued beacon leaves within the batch window; give one time to land.
  await page.waitForTimeout(1500);
}

test("Global Privacy Control and data-tracking=off send nothing", async ({ browser, request }) => {
  await resetSignals(request);

  // The browser-level opt-out: the tracker must stay off for the whole page.
  const gpc = await browser.newContext();
  await gpc.addInitScript(() => {
    Object.defineProperty(navigator, "globalPrivacyControl", { get: () => true, configurable: true });
  });
  const gpcPage = await gpc.newPage();
  await gpcPage.goto(`${E2E}/tracking`, { waitUntil: "load" });
  expect(await gpcPage.evaluate(() => window.cqrsTracker.enabled())).toBe(false);
  await interactAndFlush(gpcPage);
  await gpc.close();

  // The page-level opt-out: <html data-tracking="off"> on the page itself.
  const page = await browser.newContext().then((context) => context.newPage());
  await page.goto(`${E2E}/tracking/off`, { waitUntil: "load" });
  expect(await page.evaluate(() => window.cqrsTracker.enabled())).toBe(false);
  await interactAndFlush(page);

  expect(await signals(request)).toEqual([]);
});
